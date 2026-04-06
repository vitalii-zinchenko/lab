# API Key Auth — DB Schema & Endpoints

## Context

We need to allow users to authenticate via API keys using the OAuth 2.0 Client Credentials Grant (RFC 6749 §4.4). Each user can have multiple API keys (client_id + client_secret pairs). Exchanging valid credentials returns a signed JWT, which is then validated by middleware on all protected endpoints.

---

## Answers to Open Questions

### How to store the client secret?
Store a **bcrypt hash** of the secret (cost factor 12). Bcrypt is the pragmatic Go choice (`golang.org/x/crypto/bcrypt` ships with the standard crypto library). Argon2id is more modern (recommended by OWASP 2026) but requires more configuration and is overkill for a lab. Show the plain-text secret **once** at creation time — we cannot retrieve it afterward.

### JWT signing algorithm — HMAC or RSA?
**HS256 (HMAC-SHA256)** is the right choice for a single server with a symmetric key. We sign and verify with the same secret; there's no need for the asymmetric key distribution that RS256 solves. HS256 is simpler, faster, and well-supported.

### Should the token be encrypted?
**No.** The payload only contains the user ID (not sensitive). Signing (JWS) is sufficient — it guarantees integrity and authenticity. Encryption (JWE) is only needed when the payload itself is sensitive.

---

## DB Schema

### New migration: `00004_create_api_keys_table.sql`

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS api_keys (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           INTEGER      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id         UUID         NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    client_secret_hash VARCHAR(255) NOT NULL,
    name              VARCHAR(255),                      -- human-readable label (e.g. "CI pipeline")
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expires_at        TIMESTAMPTZ,                       -- NULL = never expires
    revoked_at        TIMESTAMPTZ,                       -- NULL = active; set to revoke
    last_used_at      TIMESTAMPTZ
);

CREATE INDEX idx_api_keys_client_id ON api_keys(client_id);
CREATE INDEX idx_api_keys_user_id   ON api_keys(user_id);

-- +goose Down
DROP TABLE IF EXISTS api_keys;
```

**Notes:**
- `client_id` is a UUID stored in plain text — safe to expose.
- `client_secret_hash` stores the bcrypt hash. The raw secret is returned once at creation and never stored.
- `revoked_at` enables soft-delete revocation without losing audit history.
- `expires_at` allows time-limited keys.

---

## APIs

### 1. `POST /token` — Exchange credentials for JWT

OAuth Client Credentials Grant (RFC 6749 §4.4).

**Request:**
```
POST /token
Content-Type: application/x-www-form-urlencoded

grant_type=client_credentials&client_id=<uuid>&client_secret=<raw_secret>
```

**Success Response `200`:**
```json
{
  "access_token": "<jwt>",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

**Error Responses:** `400` (missing fields), `401` (invalid credentials or revoked key), `401` (expired key).

**Logic:**
1. Look up `api_keys` by `client_id`.
2. Check `revoked_at IS NULL` and `(expires_at IS NULL OR expires_at > NOW())`.
3. `bcrypt.CompareHashAndPassword(stored_hash, presented_secret)`.
4. On success, issue JWT; update `last_used_at`.

**JWT Claims:**
```json
{
  "sub":  "42",          // user_id as string
  "iss":  "lab-api",
  "iat":  1712345678,
  "exp":  1712349278,    // iat + 3600s (1 hour)
  "jti":  "<uuid>"       // for future revocation / replay prevention
}
```

---

### 2. `POST /users` — Create a new user

Unauthenticated. Required before creating API keys.

**Request:**
```json
{ "username": "alice", "email": "alice@example.com" }
```

**Success Response `201`:**
```json
{ "id": 1, "username": "alice", "email": "alice@example.com", "created_at": "..." }
```

**Logic:** Insert into existing `users` table. Return the `id` — the caller uses it to create API keys.

---

### 3. `POST /api-keys` — Create a new API key

Unauthenticated for now. Requires a valid `user_id` (obtained from `POST /users`).

**Request:**
```json
{ "user_id": 1, "name": "CI pipeline", "expires_at": "2027-01-01T00:00:00Z" }
```

**Success Response `201`:**
```json
{
  "client_id":     "<uuid>",
  "client_secret": "<raw-secret>",   // shown ONCE — store securely
  "name":          "CI pipeline",
  "expires_at":    "2027-01-01T00:00:00Z",
  "created_at":    "2026-04-06T10:00:00Z"
}
```

**Logic:** Generate UUID for `client_id`, generate 32-byte random secret, bcrypt-hash it, insert row. Return both in plain text (only opportunity to do so).

---

### 4. `DELETE /api-keys/:client_id` — Revoke a key

Sets `revoked_at = NOW()`. Protected by JWT middleware (user can only revoke their own keys).

**Success Response:** `204 No Content`

---

### 5. `GET /api-keys` — List keys for the authenticated user

Returns all keys (sans hashes) for the user identified in the JWT. Protected by JWT middleware.

**Success Response `200`:**
```json
[
  {
    "client_id":   "<uuid>",
    "name":        "CI pipeline",
    "created_at":  "...",
    "expires_at":  null,
    "revoked_at":  null,
    "last_used_at": "..."
  }
]
```

---

## Auth Middleware

Gin middleware (`services/server/api/middleware/auth.go`):

1. Read `Authorization: Bearer <token>` header — return `401` if missing.
2. Parse and verify JWT signature using the shared HMAC secret (env var `JWT_SECRET`).
3. Validate `exp` claim — return `401` if expired.
4. Extract `sub` (user_id) — set `c.Set("user_id", userID)`.
5. Call `c.Next()`.

Applied selectively to protected routes (not `/token`, `/health`).

---

## New Files / Changed Files

| File | Action |
|---|---|
| `infra/migrations/00004_create_api_keys_table.sql` | New migration |
| `services/server/model/api_key.go` | New GORM model |
| `services/server/repository/api_key.go` | New repository (interface + GORM impl) |
| `services/server/model/user.go` | New GORM model for users |
| `services/server/repository/user.go` | New repository (interface + GORM impl) |
| `services/server/api/users.go` | New — `POST /users` handler |
| `services/server/api/auth.go` | New — `/token` handler and `/api-keys` CRUD |
| `services/server/api/middleware/auth.go` | New — JWT validation middleware |
| `services/server/api/handler.go` | Embed new ApiKeyHandler |
| `services/server/api/openapi.yaml` | Add new endpoints |
| `services/server/cmd/server/main.go` | Register middleware + new routes |
| `services/server/go.mod` | Add `golang-jwt/jwt/v5`, `golang.org/x/crypto` |

---

## Libraries

- **JWT:** `github.com/golang-jwt/jwt/v5` (most maintained Go JWT lib)
- **Bcrypt:** `golang.org/x/crypto/bcrypt` (already in crypto family, no new dep needed if `x/crypto` is transitively present — otherwise add it)
- **Random secret generation:** `crypto/rand` from stdlib (already available)

---

## Verification / Testing

1. Run migration: `make db-migrate-up`
2. Create a user: `POST /users` → save `id`
3. Create an API key: `POST /api-keys` with `user_id` → save `client_secret` from response
4. Exchange for token: `POST /token` with `client_id` + `client_secret` → get JWT
4. Decode JWT at jwt.io — verify `sub` = user_id, `exp` in future
5. Call a protected endpoint with `Authorization: Bearer <token>` → `200`
6. Call with expired/wrong token → `401`
7. Revoke the key: `DELETE /api-keys/:client_id` → `204`
8. Try to get a new token with revoked key → `401`
