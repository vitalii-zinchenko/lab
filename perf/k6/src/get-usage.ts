/**
 * get-usage — read load against GET /usage (cursor-paginated raw records).
 *
 * Each VU is assigned a fixed mock user based on its VU number. Tokens are
 * fetched lazily on the first iteration and refreshed automatically before
 * they expire — no slow setup() pre-authentication.
 *
 * Build & run:
 *   cd perf/k6 && npm run get-usage
 *   BASE_URL=http://staging:8080 npm run get-usage
 *
 * Generate credentials first:
 *   make mock-create-api-keys
 */
import { check } from 'k6';
import { UserCredential, getOrRefreshToken, getUsage } from './lib/api';

const CREDENTIALS: UserCredential[] = JSON.parse(open('../../mock-data/users-credentials.json'));

// Each VU gets a deterministic, fixed user for the lifetime of the test.
// __VU is 1-indexed so subtract 1 before modulo.
const MY_CRED = CREDENTIALS[(__VU - 1) % CREDENTIALS.length];

export const options = {
  scenarios: {
    constant_rps: {
      executor: 'constant-arrival-rate',
      rate: 1000,
      timeUnit: '1s',
      duration: '876000h', // run until stopped (Ctrl+C)
      preAllocatedVUs: 20,
      maxVUs: 200,
    },
  },
  thresholds: {
    http_req_failed:                         ['rate<0.01'],
    'http_req_duration{endpoint:get_usage}': ['p(95)<500'],
  },
};

const TWO_YEARS_MS  = 2 * 365 * 24 * 60 * 60 * 1000;
const THIRTY_DAYS_MS = 30 * 24 * 60 * 60 * 1000;

function randomWindow(): { from: string; to: string } {
  const now = Date.now();
  const fromMs = now - TWO_YEARS_MS + Math.random() * (TWO_YEARS_MS - THIRTY_DAYS_MS);
  return {
    from: new Date(fromMs).toISOString(),
    to:   new Date(fromMs + THIRTY_DAYS_MS).toISOString(),
  };
}

export default function (): void {
  const token = getOrRefreshToken(MY_CRED);
  const { from, to } = randomWindow();
  const res = getUsage(from, to, 100, null, token);
  check(res, { 'get_usage: 200': (r) => r.status === 200 });
}
