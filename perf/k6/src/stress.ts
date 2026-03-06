/**
 * Stress test — push the server to its breaking point.
 *
 * Ramps aggressively to 200 VUs while keeping the scenario tight (no sleep
 * between requests) to maximise concurrency pressure.  Thresholds are
 * intentionally relaxed: the goal here is to observe *where* the server
 * degrades rather than to enforce strict SLOs.
 *
 * Watch for:
 *   - Error rate climbing above 5%
 *   - p(95) latency exceeding 2 s
 *   - Status codes other than 2xx / 404
 *
 * Build & run:
 *   cd perf/k6 && npm run stress
 *   BASE_URL=http://staging:8080 npm run stress
 */
import { check, sleep } from 'k6';
import { createItem, deleteItem, getItem, listItems } from './lib/api';
import { randomDescription, randomItemName } from './lib/helpers';

export const options = {
  stages: [
    { duration: '1m',  target: 1000  },  // warm up
    { duration: '2m',  target: 1000 },  // moderate pressure
    { duration: '20m',  target: 1000 },  // heavy pressure
    { duration: '1m',  target: 0   },  // recovery
  ],
  thresholds: {
    // Relaxed — we want to observe, not just fail fast
    http_req_failed:   ['rate<0.05'],    // alert if > 5% requests fail
    http_req_duration: ['p(95)<2000'],   // alert if 95th percentile > 2 s
  },
};

export default function (): void {
  const created = createItem(randomItemName(), randomDescription());
  if (created.status !== 201) {
    // Under extreme load the server may reject; count it and move on
    sleep(0.2);
    return;
  }

  const id = created.json('id') as string;

  check(getItem(id),    { 'get: 200':    (r) => r.status === 200 });
  check(listItems(),    { 'list: 200':   (r) => r.status === 200 });
  check(deleteItem(id), { 'delete: 204': (r) => r.status === 204 });

  // Minimal think time so VUs keep hammering
  sleep(Math.random() * 0.3);
}
