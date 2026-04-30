package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/golang-jwt/jwt/v5"

	"github.com/antongolenev23/tuchka-server/internal/auth"
	"github.com/antongolenev23/tuchka-server/internal/config"
	"github.com/antongolenev23/tuchka-server/pkg/api/response"
)

type contextKey string

const UserIDKey contextKey = "userID"

func AuthMiddleware(cfg *config.Config, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		const op = "middleware.auth"

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.With(
				slog.String("op", op),
				slog.String("request_id", middleware.GetReqID(r.Context())),
			)

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				log.Info("no Authorization header")
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, response.Error("unauthorized"))
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == authHeader {
				log.Info("invalid Authorization header",
					slog.String("auth_header", authHeader),
				)
				render.Status(r, http.StatusUnauthorized)
				return
			}

			claims := &auth.Claims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
				if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(cfg.Auth.JWTSecret), nil
			})
			if err != nil {
				log.Info("failed to parse token")
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, response.Error("unauthorized"))
				return
			} else if !token.Valid {
				log.Info("invalid authorization token")
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, response.Error("unauthorized"))
				return
			}

			log.Info("authorization completed")

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
