package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
	"{{projectName}}/db"
	"{{projectName}}/service"

	"golang.org/x/time/rate"
)

type userctx string

const userKey userctx = "user"

type middleware func(http.Handler) http.Handler

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func Use(handler http.Handler, mw ...middleware) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		handler = mw[i](handler)
	}
	return handler
}

func rateLimitMiddlewarePerIP(r rate.Limit, b int) middleware {
	visitors := make(map[string]*rate.Limiter)
	var mu sync.Mutex

	go func() {
		for {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			visitors = make(map[string]*rate.Limiter)
			mu.Unlock()
		}
	}()

	getVisitor := func(ip string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()
		if limiter, exists := visitors[ip]; exists {
			return limiter
		}
		limiter := rate.NewLimiter(r, b)
		visitors[ip] = limiter
		return limiter
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if colon := strings.LastIndex(ip, ":"); colon != -1 {
				ip = ip[:colon]
			}

			if !getVisitor(ip).Allow() {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func loggingMiddleware(logger *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(lrw, r)
			duration := time.Since(start)

			logger.Info("http request handled",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", lrw.statusCode),
				slog.Duration("duration", duration))
		})
	}
}

func authMiddleware(store db.Store, logger *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionCookie, err := r.Cookie("session")
			if err != nil || sessionCookie.Value == "" {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			session, err := store.GetByCookieHash(r.Context(), sessionCookie.Value)
			if err != nil {
				logger.Error("failed to get session by hash", slog.String("error", err.Error()))
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			user, err := store.GetUserByID(r.Context(), session.UserID)
			if err != nil || user == nil {
				logger.Error("failed to get user by ID", slog.String("error", err.Error()))
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			ctx := context.WithValue(r.Context(), userKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers := w.Header()
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("Content-Security-Policy", "script-src 'self'; style-src 'self';")
		headers.Set("X-Frame-Options", "DENY")
		headers.Set("X-XSS-Protection", "1; mode=block")
		headers.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		headers.Set("Cache-Control", "no-store, max-age=0")
		headers.Set("Pragma", "no-cache")
		if r.TLS != nil {
			headers.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

func sessionRefreshMiddleware(store db.Store) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cookie, err := r.Cookie("session"); err == nil {
				if err := service.RefreshSession(r.Context(), cookie.Value, store); err == nil {
					http.SetCookie(w, &http.Cookie{
						Name:     "session",
						Value:    cookie.Value,
						Path:     "/",
						HttpOnly: true,
						Secure:   true,
						SameSite: http.SameSiteStrictMode,
						MaxAge:   int(24 * time.Hour.Seconds()),
					})
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func onlyAdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := contextUser(r)
		if user == nil || user.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func mustBeVerifyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := contextUser(r)
		if u == nil {
			unauthorized(w)
			return
		}

		if !u.Verify && r.URL.Path != "/verify" {
			unauthorized(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func contextUser(r *http.Request) *db.User {
	if user, ok := r.Context().Value(userKey).(*db.User); ok {
		return user
	}
	return nil
}

func AllRouteMiddleware(logger *slog.Logger) []middleware {
	return []middleware{
		loggingMiddleware(logger),
		securityHeadersMiddleware,
		rateLimitMiddlewarePerIP(rate.Every(time.Second), 10),
	}
}

func AdminMiddleware(store db.Store, logger *slog.Logger) []middleware {
	return []middleware{
		sessionRefreshMiddleware(store),
		authMiddleware(store, logger),
		onlyAdminMiddleware,
		mustBeVerifyMiddleware,
	}
}
