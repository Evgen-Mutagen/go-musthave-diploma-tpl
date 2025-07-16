package middlewareinternal

import (
	"context"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/service"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/types"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/util/logger"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

func JWTAuthMiddleware(authService service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, err := extractToken(r)
			if err != nil {
				logger.Log.Debug("Failed to extract token",
					zap.String("path", r.URL.Path),
					zap.Error(err))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			userID, err := authService.ValidateToken(tokenString)
			if err != nil {
				logger.Log.Warn("Invalid token",
					zap.String("path", r.URL.Path),
					zap.Error(err))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), types.UserIDKey, userID)
			logger.Log.Debug("User authenticated",
				zap.Int64("user_id", userID),
				zap.String("path", r.URL.Path))

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractToken(r *http.Request) (string, error) {
	cookie, err := r.Cookie("jwt")
	if err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", http.ErrNoCookie
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", http.ErrNoCookie
	}

	return parts[1], nil
}

func GetUserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(types.UserIDKey).(int64)
	return userID, ok
}
