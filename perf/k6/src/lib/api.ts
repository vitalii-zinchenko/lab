import http, { Params } from 'k6/http';
import { BASE_URL, JSON_HEADERS } from './helpers';

// Domain types matching the server's OpenAPI schema
export interface Item {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
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
