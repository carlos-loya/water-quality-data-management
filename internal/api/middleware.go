package api

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/carlos-loya/water-quality-data-management/internal/auth"
)

// withLogging wraps an http.Handler and logs each request.
func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration", time.Since(start),
		)
	})
}

// withAuth validates the JWT from the Authorization header and injects user claims into the context.
// Requests without a valid token receive 401.
func withAuth(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing or invalid authorization header")
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")

		claims, err := auth.ValidateToken(secret, tokenStr)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		ctx := auth.WithUser(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireRole wraps a handler and returns 403 if the user lacks the required role.
func requireRole(role string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := auth.UserFrom(r.Context())
		if claims == nil || !claims.HasRole(role) {
			writeError(w, http.StatusForbidden, "insufficient permissions")
			return
		}
		next(w, r)
	}
}

// requireAnyRole wraps a handler and returns 403 if the user holds none of the listed roles.
func requireAnyRole(roles []string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := auth.UserFrom(r.Context())
		if claims == nil || !claims.HasAnyRole(roles...) {
			writeError(w, http.StatusForbidden, "insufficient permissions")
			return
		}
		next(w, r)
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
