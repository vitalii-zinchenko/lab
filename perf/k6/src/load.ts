/**
 * Load test — ramp up to 50 VUs over 4 minutes, then cool down.
 *
 * Models realistic concurrent usage: each VU creates an item, reads it,
 * lists items, then deletes it — with a short think-time in between.
 *
 * Per-endpoint thresholds fail the test when individual SLOs are breached.
 *
 * Build & run:
 *   cd perf/k6 && npm run load
 *   BASE_URL=http://staging:8080 npm run load
 */
import { check, sleep } from 'k6';
import { healthCheck, listItems, createItem, getItem, deleteItem } from './lib/api';
import { randomItemName, randomDescription } from './lib/helpers';

export const options = {
  stages: [
    { duration: '30s', target: 10 },  // warm up
    { duration: '1m',  target: 50 },  // ramp to peak
    { duration: '2m',  target: 50 },  // hold peak
    { duration: '30s', target: 0 },   // cool down
  ],
  thresholds: {
    // Global
    http_req_failed:   ['rate<0.01'],   // < 1% total error rate
    http_req_duration: ['p(95)<500'],   // 95th percentile under 500 ms

    // Per-endpoint (tagged in api.ts)
    'http_req_duration{endpoint:health}':      ['p(95)<50'],
    'http_req_duration{endpoint:list_items}':  ['p(95)<200'],
    'http_req_duration{endpoint:create_item}': ['p(95)<300'],
    'http_req_duration{endpoint:get_item}':    ['p(95)<100'],
    'http_req_duration{endpoint:delete_item}': ['p(95)<100'],
  },
};

export default function (): void {
  // Keep the health endpoint warm alongside the CRUD traffic
  healthCheck();

  // CRUD lifecycle
  const created = createItem(randomItemName(), randomDescription());
  if (created.status !== 201) {
    sleep(1);
    return;
  }
  const id = created.json('id') as string;

  check(getItem(id), { 'get: 200': (r) => r.status === 200 });

  // Mix of list calls — some with a limit, some without
  listItems();
  if (Math.random() < 0.5) {
    listItems(10);
  }

  sleep(0.5); // brief pause to simulate a user reading the list

  check(deleteItem(id), { 'delete: 204': (r) => r.status === 204 });

  // Think time: 0.5 – 2 s (simulates user processing time between actions)
  sleep(Math.random() * 1.5 + 0.5);
}
