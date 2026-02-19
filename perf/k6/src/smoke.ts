/**
 * Smoke test — 1 VU, 1 minute.
 *
 * Covers the full CRUD lifecycle once per iteration to confirm every endpoint
 * works correctly before running heavier load tests.
 *
 * Build & run:
 *   cd perf/k6 && npm run smoke
 *   BASE_URL=http://staging:8080 npm run smoke
 */
import { check, sleep } from 'k6';
import { healthCheck, listItems, createItem, getItem, deleteItem } from './lib/api';
import { randomItemName, randomDescription } from './lib/helpers';

export const options = {
  vus: 1,
  duration: '1m',
  thresholds: {
    http_req_failed: ['rate<0.01'],     // virtually no failures allowed
    http_req_duration: ['p(95)<500'],   // every call under 500 ms
  },
};

export default function (): void {
  // 1. Health check
  const health = healthCheck();
  check(health, {
    'health: status 200': (r) => r.status === 200,
    'health: body ok':    (r) => (r.json('status') as string) === 'ok',
  });

  // 2. Create an item
  const created = createItem(randomItemName(), randomDescription());
  check(created, { 'create: status 201': (r) => r.status === 201 });
  if (created.status !== 201) {
    sleep(1);
    return;
  }
  const id = created.json('id') as string;

  // 3. Fetch the item and verify the payload
  const got = getItem(id);
  check(got, {
    'get: status 200':    (r) => r.status === 200,
    'get: correct id':    (r) => (r.json('id') as string) === id,
    'get: has name':      (r) => Boolean(r.json('name')),
    'get: has createdAt': (r) => r.json('createdAt') !== null,
  });

  // 4. List items (default + with limit)
  const listed = listItems();
  check(listed, { 'list: status 200': (r) => r.status === 200 });

  const listedLimit = listItems(5);
  check(listedLimit, {
    'list(limit=5): status 200': (r) => r.status === 200,
    'list(limit=5): at most 5':  (r) => {
      const body = r.json() as unknown[];
      return Array.isArray(body) && body.length <= 5;
    },
  });

  // 5. Delete the item
  const deleted = deleteItem(id);
  check(deleted, { 'delete: status 204': (r) => r.status === 204 });

  // 6. Verify it is truly gone
  const gone = getItem(id);
  check(gone, { 'get deleted: status 404': (r) => r.status === 404 });

  sleep(1);
}
