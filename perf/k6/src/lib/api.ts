import http, { Params } from 'k6/http';
import encoding from 'k6/encoding';
import { BASE_URL, JSON_HEADERS } from './helpers';

// Domain types matching the server's OpenAPI schema
export interface Item {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
}

export interface UserCredential {
  user_id: number;
  client_id: string;
  client_secret: string;
}

// Each function tags its request with { endpoint: '<name>' } so thresholds
// can be applied per-endpoint in the test options.

export function healthCheck() {
  return http.get(`${BASE_URL}/health`, {
    tags: { endpoint: 'health' },
  });
}

export function listItems(limit?: number) {
  const url =
    limit != null ? `${BASE_URL}/items?limit=${limit}` : `${BASE_URL}/items`;
  return http.get(url, {
    tags: { endpoint: 'list_items' },
  });
}

export function createItem(name: string, description?: string) {
  const body: Record<string, string> = { name };
  if (description != null) body.description = description;
  return http.post(`${BASE_URL}/items`, JSON.stringify(body), {
    headers: JSON_HEADERS,
    tags: { endpoint: 'create_item' },
  });
}

export function getItem(id: string, extraParams?: Params) {
  return http.get(`${BASE_URL}/items/${id}`, {
    tags: { endpoint: 'get_item' },
    ...extraParams,
  });
}

export function deleteItem(id: string) {
  return http.del(`${BASE_URL}/items/${id}`, null, {
    tags: { endpoint: 'delete_item' },
  });
}

export function createUsage(userId: number, operation: string, timestamp: string) {
  return http.post(
    `${BASE_URL}/usage`,
    JSON.stringify({ user_id: userId, operation, timestamp }),
    {
      headers: JSON_HEADERS,
      tags: { endpoint: 'create_usage' },
    },
  );
}

// ---------------------------------------------------------------------------
// Token management — lazy per-VU cache with automatic refresh
// ---------------------------------------------------------------------------

// Module-level vars are per-VU in k6 (each VU has its own JS runtime).
let _cachedToken = '';
let _tokenExpSec = 0;

// Parses the `exp` claim from a JWT payload without verifying the signature.
// Safe to use here because we just need the expiry time, not trust the token.
function parseJwtExp(token: string): number {
  try {
    const payload = JSON.parse(encoding.b64decode(token.split('.')[1], 'rawurl', 's') as string);
    return (payload.exp as number) ?? 0;
  } catch (_) {
    // Fallback: treat as expired after 1 hour from now
    return Math.floor(Date.now() / 1000) + 3600;
  }
}

// getToken exchanges client credentials for a JWT bearer token.
export function getToken(clientId: string, clientSecret: string): string {
  const res = http.post(
    `${BASE_URL}/token`,
    JSON.stringify({ grant_type: 'client_credentials', client_id: clientId, client_secret: clientSecret }),
    {
      headers: JSON_HEADERS,
      tags: { endpoint: 'get_token' },
    },
  );
  if (res.status !== 200) {
    throw new Error(`getToken failed: ${res.status} ${res.body}`);
  }
  return (res.json() as { access_token: string }).access_token;
}

// getOrRefreshToken returns a valid bearer token for the given credential,
// fetching or refreshing it transparently when it is missing or within 30s of
// expiry. Because module-level state is per-VU in k6, each VU independently
// maintains its own cached token — no cross-VU contention.
export function getOrRefreshToken(cred: UserCredential): string {
  const nowSec = Math.floor(Date.now() / 1000);
  if (_cachedToken && _tokenExpSec - 30 > nowSec) {
    return _cachedToken;
  }
  _cachedToken = getToken(cred.client_id, cred.client_secret);
  _tokenExpSec = parseJwtExp(_cachedToken);
  return _cachedToken;
}

// ---------------------------------------------------------------------------
// Usage endpoints
// ---------------------------------------------------------------------------

// getUsage fetches one page of raw usage records for the authenticated user.
// Pass cursor=null for the first page; use next_cursor from the response for
// subsequent pages.
export function getUsage(
  from: string,
  to: string,
  limit: number,
  cursor: string | null,
  token: string,
) {
  let url = `${BASE_URL}/usage?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}&limit=${limit}`;
  if (cursor != null) url += `&cursor=${cursor}`;
  return http.get(url, {
    headers: { Authorization: `Bearer ${token}`, ...JSON_HEADERS },
    tags: { endpoint: 'get_usage' },
  });
}

// getUsageStats fetches daily aggregated usage stats for the authenticated user.
export function getUsageStats(from: string, to: string, token: string) {
  const url = `${BASE_URL}/usage/stats?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`;
  return http.get(url, {
    headers: { Authorization: `Bearer ${token}`, ...JSON_HEADERS },
    tags: { endpoint: 'get_usage_stats' },
  });
}
