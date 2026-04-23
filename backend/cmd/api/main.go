package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/jf-ar/compli-church/internal/adapters/postgres"
	"github.com/jf-ar/compli-church/internal/config"
	"github.com/jf-ar/compli-church/internal/handlers"
	"github.com/jf-ar/compli-church/internal/services"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	privateKey, err := loadPrivateKey(cfg.JWTPrivateKeyPath)
	if err != nil {
		slog.Error("failed to load JWT private key", "path", cfg.JWTPrivateKeyPath, "error", err)
		os.Exit(1)
	}
	publicKey, err := loadPublicKey(cfg.JWTPublicKeyPath)
	if err != nil {
		slog.Error("failed to load JWT public key", "path", cfg.JWTPublicKeyPath, "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Wire up adapters, services, and handlers
	authRepo := postgres.NewAuthRepo(pool)
	authSvc := services.NewAuthService(authRepo, privateKey, publicKey, cfg.JWTAccessTTL, cfg.JWTRefreshTTLDays)
	authHandler := handlers.NewAuthHandler(authSvc, cfg.JWTRefreshTTLDays, cfg.Env == "production")

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	r.Route("/api/v1", func(r chi.Router) {
		// ── Public auth endpoints ─────────────────────────────────────────
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.Refresh)

		// ── Protected endpoints ───────────────────────────────────────────
		r.Group(func(r chi.Router) {
			r.Use(handlers.Authenticate(authSvc))

			r.Post("/auth/logout", authHandler.Logout)
			r.Post("/auth/logout-all", authHandler.LogoutAll)

			// Additional domain routes will be mounted here as they are implemented:
			// r.Mount("/members",   membersRouter(memberHandler))
			// r.Mount("/churches",  churchesRouter(churchHandler))
			// r.Mount("/schedules", schedulesRouter(scheduleHandler))
			// r.Mount("/inventory", inventoryRouter(inventoryHandler))
		})
	})

	addr := ":" + cfg.Port
	slog.Info("server starting", "addr", addr, "env", cfg.Env)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server exited", "error", err)
		os.Exit(1)
	}
}

// ── key loaders ───────────────────────────────────────────────────────────────

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in %s", path)
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("PKCS8 key is not RSA")
		}
		return rsaKey, nil
	default:
		return nil, fmt.Errorf("unsupported PEM block type: %s", block.Type)
	}
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in %s", path)
	}
	switch block.Type {
	case "RSA PUBLIC KEY":
		return x509.ParsePKCS1PublicKey(block.Bytes)
	case "PUBLIC KEY":
		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rsaKey, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("PKIX key is not RSA")
		}
		return rsaKey, nil
	default:
		return nil, fmt.Errorf("unsupported PEM block type: %s", block.Type)
	}
}
