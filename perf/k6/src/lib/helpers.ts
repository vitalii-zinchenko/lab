// Base URL can be overridden at runtime:  BASE_URL=http://staging:8080 k6 run dist/smoke.js
export const BASE_URL: string = __ENV.BASE_URL ?? 'http://localhost:8080';

export const JSON_HEADERS = { 'Content-Type': 'application/json' } as const;

const CHARS = 'abcdefghijklmnopqrstuvwxyz0123456789';

export function randomString(length = 8): string {
  let s = '';
  for (let i = 0; i < length; i++) {
    s += CHARS[Math.floor(Math.random() * CHARS.length)];
  }
  return s;
}

export function randomItemName(): string {
  return `item-${randomString(8)}`;
}

export function randomDescription(): string {
  return `desc-${randomString(12)}`;
}
