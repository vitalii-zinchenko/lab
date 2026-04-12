/**
 * Insert-usage perf script — constant 100 RPS against POST /usage.
 *
 * Uses a fixed user ID (3) and randomises the operation name so rows
 * aren't trivially deduplicated.
 *
 * Build & run:
 *   cd perf/k6 && npm run insert-usage
 *   BASE_URL=http://staging:8080 npm run insert-usage
 */
import { check } from 'k6';
import { createUsage } from './lib/api';

const USER_ID = 3;

const OPERATIONS = ['read', 'write', 'delete', 'update', 'export'];

export const options = {
  scenarios: {
    constant_rps: {
      executor: 'constant-arrival-rate',
      rate: 10000,
      timeUnit: '1s',
      duration: '876000h', // run until stopped (Ctrl+C)
      preAllocatedVUs: 20,  // initial pool; k6 scales up if needed
      maxVUs: 100,
    },
  },
  thresholds: {
    http_req_failed:                              ['rate<0.01'],   // < 1% errors
    'http_req_duration{endpoint:create_usage}':  ['p(95)<300'],   // 95th p < 300 ms
  },
};

export default function (): void {
  const operation = OPERATIONS[Math.floor(Math.random() * OPERATIONS.length)];
  const timestamp = new Date().toISOString();

  const res = createUsage(USER_ID, operation, timestamp);

  check(res, { 'create_usage: 201': (r) => r.status === 201 });
}
