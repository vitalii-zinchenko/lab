package handlers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/golang-jwt/jwt/v5"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
)

// contextKey is an unexported type for context keys in this package.
type contextKey string

// UserIDKey is the context key under which the authenticated user's ID is stored.
const UserIDKey contextKey = "user_id"

// GetUserID extracts the authenticated user ID from a request context.
// Returns (0, false) if no authenticated user is present.
func GetUserID(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(UserIDKey).(int64)
	return id, ok
}

// NewAuthenticationFunc returns an openapi3filter.AuthenticationFunc that validates
// JWT Bearer tokens for routes marked with security requirements in the OpenAPI spec.
//
// On success, the authenticated user's ID is stored in both the Gin context and
// the request context so downstream handlers can retrieve it via GetUserID.
func NewAuthenticationFunc(jwtSecret []byte) openapi3filter.AuthenticationFunc {
	return func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
		gCtx := ginmiddleware.GetGinContext(ctx)

		authHeader := gCtx.GetHeader("Authorization")
		if authHeader == "" {
			return errors.New("missing authorization header")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return errors.New("invalid authorization header format")
		}

		token, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return jwtSecret, nil
		}, jwt.WithValidMethods([]string{"HS256"}))

		if err != nil || !token.Valid {
			return errors.New("invalid or expired token")
		}

		sub, err := token.Claims.GetSubject()
		if err != nil {
			return errors.New("invalid token subject")
		}

		userID, err := strconv.ParseInt(sub, 10, 64)
		if err != nil {
			return errors.New("invalid token subject")
		}

		gCtx.Set("user_id", userID)
		reqCtx := context.WithValue(gCtx.Request.Context(), UserIDKey, userID)
		gCtx.Request = gCtx.Request.WithContext(reqCtx)

		return nil
	}
}
