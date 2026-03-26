package magiclink

import (
	"context"
	"net/http"
)

type contextKey string

const (
	userIDContextKey contextKey = "magiclink_user_id"
	claimsContextKey contextKey = "magiclink_claims"
)

// UserIDFromContext returns the authenticated user id set by Middleware.
func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDContextKey).(string)
	return v, ok && v != ""
}

// ClaimsFromContext returns claims set by Middleware.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	v, ok := ctx.Value(claimsContextKey).(*Claims)
	return v, ok && v != nil
}

// Middleware authenticates Bearer JWTs, upserts user identity, and stores auth context.
func (s *Service) Middleware(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, claims, err := s.AuthenticateBearer(r.Context(), r.Header.Get("Authorization"))
		if err != nil {
			writeError(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), userIDContextKey, userID)
		ctx = context.WithValue(ctx, claimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
