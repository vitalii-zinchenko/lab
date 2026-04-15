/**
 * insert-usage — constant-RPS write load against POST /usage.
 *
 * Picks a random mock user from the credentials file on each iteration
 * (no auth required for POST /usage).
 *
 * Build & run:
 *   cd perf/k6 && npm run insert-usage
 *   BASE_URL=http://staging:8080 npm run insert-usage
 *
 * Generate credentials first:
 *   make mock-create-api-keys
 */
import { check } from 'k6';
import { UserCredential, createUsage } from './lib/api';

// Load credentials file written by `mock-data create-api-keys`.
// open() is a k6 global available only in the init context; it reads the file
// as a string synchronously before any VU starts.
// Path is relative to this script file.
const CREDENTIALS: UserCredential[] = JSON.parse(open('../../mock-data/users-credentials.json'));

const OPERATIONS = ['read', 'write', 'delete', 'create'];

export const options = {
  scenarios: {
    constant_rps: {
      executor: 'constant-arrival-rate',
      rate: 100,
      timeUnit: '1s',
      duration: '876000h', // run until stopped (Ctrl+C)
      preAllocatedVUs: 20,
      maxVUs: 100,
    },
  },
  thresholds: {
    http_req_failed:                              ['rate<0.01'],
    'http_req_duration{endpoint:create_usage}':  ['p(95)<300'],
  },
};

export default function (): void {
  const cred = CREDENTIALS[Math.floor(Math.random() * CREDENTIALS.length)];
  const operation = OPERATIONS[Math.floor(Math.random() * OPERATIONS.length)];
  const timestamp = new Date().toISOString();

  const res = createUsage(cred.user_id, operation, timestamp);

  check(res, { 'create_usage: 201': (r) => r.status === 201 });
}
