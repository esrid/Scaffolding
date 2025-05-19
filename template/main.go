package main

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"{{projectName}}/config"
	"{{projectName}}/db"
	"{{projectName}}/handler"
	"{{projectName}}/web"
)

func main() {
	cfg := config.Load()
	logger := config.NewSlog(cfg.Env)

	conn, err := db.NewDB(cfg.Database.String())
	if err != nil {
		logger.Error("unable to connect to database", slog.String("error", err.Error()))
		return
	}
	defer conn.Close()

	if err := db.MigrateSchema(conn, logger); err != nil {
		logger.Error("unable to perform migration", slog.String("error", err.Error()))
		return
	}

	if err := db.CreateSeed(conn); err != nil {
		logger.Error("unable to seed admin data", slog.String("error", err.Error()))
		return
	}

	r := newRouter(logger, db.NewPostgresStore(conn), cfg.Google, cfg.MAIlAPI)
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r.route(),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("starting server", slog.String("port", cfg.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("unable to start server", slog.String("error", err.Error()))
		}
	}()

	<-ctx.Done()
	stop()
	logger.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", slog.String("error", err.Error()))
	} else {
		logger.Info("server stopped gracefully")
	}
}

type router struct {
	logger *slog.Logger
	store  *db.PostgresStore
	google *config.GoogleOAuth
	mail   string
}

func newRouter(logger *slog.Logger, store *db.PostgresStore, google *config.GoogleOAuth, mail string) *router {
	return &router{logger: logger, store: store, google: google, mail: mail}
}

func (r *router) route() http.Handler {
	mux := http.NewServeMux()
	r.setupStatic(mux)
	r.setupPublic(mux)
	r.setupAdmin(mux)

	return handler.Use(mux, handler.AllRouteMiddleware(r.logger)...)
}

func (r *router) setupStatic(mux *http.ServeMux) {
	staticFS, err := fs.Sub(web.WebFs, "static")
	if err != nil {
		r.logger.Error("unable to serve static files", slog.String("error", err.Error()))
		return
	}
	mux.Handle("/static/", http.StripPrefix("/static", http.FileServerFS(staticFS)))
}

func (r *router) setupPublic(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", handler.Home)
	mux.HandleFunc("GET /inscription", handler.GetRegister)
	mux.HandleFunc("POST /inscription", handler.RegisterUser(r.store, r.logger))
	mux.HandleFunc("GET /connexion", handler.GetLogin)
	mux.HandleFunc("POST /connexion", handler.PostLogin(r.store, r.logger))
	mux.HandleFunc("GET /auth/google/login", handler.HandleGoogleLogin(r.google.Oauth()))
	mux.HandleFunc("GET /auth/google/callback", handler.HandleGoogleCallback(r.store, r.google.Oauth(), r.logger))

	// ADMIN
	mux.HandleFunc("GET /admin/login", handler.GetAdminLogin)
	mux.HandleFunc("POST /admin/login", handler.PostAdminLogin(r.store, r.logger, r.mail))
}

func (r *router) setupAdmin(mux *http.ServeMux) {
	privateMux := http.NewServeMux()
	privateMux.HandleFunc("GET /verify", handler.GetVerifyOTP(r.store, r.logger))
	privateMux.HandleFunc("POST /verify", handler.PostVerifyOTP(r.store))
	privateMux.HandleFunc("GET /dashboard", handler.Dashboard)
	privateHandler := handler.Use(privateMux, handler.AdminMiddleware(r.store, r.logger)...)
	mux.Handle("/admin/", http.StripPrefix("/admin", privateHandler))
}
