package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// userCredential is one entry in the output JSON file.
type userCredential struct {
	UserID       int64  `json:"user_id"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// createApiKeys ensures every mock user has exactly one non-revoked API key
// named "mock_key" and a password stored in password_hash. Existing keys and
// passwords are reused when the output file already records them; otherwise
// they are recreated. The full credentials list is written to --out on success.
func createApiKeys(ctx context.Context, pool *pgxpool.Pool, outPath string) error {
	existing := map[int64]userCredential{}
	if data, err := os.ReadFile(outPath); err == nil {
		var creds []userCredential
		if json.Unmarshal(data, &creds) == nil {
			for _, c := range creds {
				existing[c.UserID] = c
			}
		}
	}

	type userRow struct {
		ID    int64
		Email string
	}
	rows, err := pool.Query(ctx,
		`SELECT id, email FROM users WHERE username LIKE $1 ORDER BY id`,
		usernamePrefix+"%",
	)
	if err != nil {
		return fmt.Errorf("query users: %w", err)
	}
	users, err := pgx.CollectRows(rows, pgx.RowToStructByPos[userRow])
	if err != nil {
		return fmt.Errorf("collect users: %w", err)
	}
	log.Printf("found %d mock users", len(users))

	result := make([]userCredential, 0, len(users))

	for _, u := range users {
		cred := userCredential{UserID: u.ID, Email: u.Email}

		// ---- password ----
		var hasPassword bool
		pool.QueryRow(ctx,
			`SELECT password_hash IS NOT NULL FROM users WHERE id = $1`,
			u.ID,
		).Scan(&hasPassword)

		if prev, ok := existing[u.ID]; ok && prev.Password != "" && hasPassword {
			cred.Password = prev.Password
		} else {
			rawPassword, err := generateSecret()
			if err != nil {
				return fmt.Errorf("generate password for user %d: %w", u.ID, err)
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("hash password for user %d: %w", u.ID, err)
			}
			if _, err := pool.Exec(ctx,
				`UPDATE users SET password_hash = $1 WHERE id = $2`,
				string(hash), u.ID,
			); err != nil {
				return fmt.Errorf("update password_hash for user %d: %w", u.ID, err)
			}
			cred.Password = rawPassword
		}

		// ---- api key ----
		var dbClientID string
		keyErr := pool.QueryRow(ctx,
			`SELECT client_id FROM api_keys
			 WHERE user_id = $1 AND name = 'mock_key' AND revoked_at IS NULL
			 LIMIT 1`,
			u.ID,
		).Scan(&dbClientID)

		keyExists := keyErr == nil

		if keyExists {
			if prev, ok := existing[u.ID]; ok && prev.ClientID == dbClientID {
				cred.ClientID = prev.ClientID
				cred.ClientSecret = prev.ClientSecret
				result = append(result, cred)
				continue
			}
			if _, err := pool.Exec(ctx,
				`DELETE FROM api_keys WHERE user_id = $1 AND name = 'mock_key'`,
				u.ID,
			); err != nil {
				return fmt.Errorf("delete stale key for user %d: %w", u.ID, err)
			}
		}

		rawSecret, err := generateSecret()
		if err != nil {
			return fmt.Errorf("generate secret for user %d: %w", u.ID, err)
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(rawSecret), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash secret for user %d: %w", u.ID, err)
		}

		clientID := uuid.New()
		now := time.Now().UTC()
		keyName := "mock_key"
		if _, err := pool.Exec(ctx,
			`INSERT INTO api_keys (id, user_id, client_id, client_secret_hash, name, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			uuid.New(), u.ID, clientID, string(hash), keyName, now,
		); err != nil {
			return fmt.Errorf("insert api key for user %d: %w", u.ID, err)
		}

		cred.ClientID = clientID.String()
		cred.ClientSecret = rawSecret
		result = append(result, cred)
	}

	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	if err := os.WriteFile(outPath, out, 0600); err != nil {
		return fmt.Errorf("write credentials file: %w", err)
	}
	log.Printf("wrote %d credentials to %s", len(result), outPath)
	return nil
}

// generateSecret produces a cryptographically random URL-safe base64 string.
func generateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
