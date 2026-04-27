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
	"github.com/go-chi/cors"

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

	memberRepo := postgres.NewMemberRepo(pool)
	memberSvc := services.NewMemberService(memberRepo, memberRepo, memberRepo, nil) // mailer wired in phase 2
	memberHandler := handlers.NewMemberHandler(memberSvc)
	roleHandler := handlers.NewRoleHandler(memberSvc)
	instrumentHandler := handlers.NewInstrumentHandler(memberSvc)

	inventoryRepo := postgres.NewInventoryRepo(pool)
	inventorySvc := services.NewInventoryService(inventoryRepo)
	inventoryHandler := handlers.NewInventoryHandler(inventorySvc)

	allowedOrigins := []string{"http://localhost:3000"}
	if cfg.Env == "production" {
		allowedOrigins = []string{"https://igreaorganizada.com.br", "https://www.igreaorganizada.com.br"}
	}

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true, // required for HttpOnly refresh-token cookie
		MaxAge:           300,
	}))
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	r.Route("/api/v1", func(r chi.Router) {
		// ── Public auth endpoints ─────────────────────────────────────────
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.Refresh)

		// ── Protected endpoints ───────────────────────────────────────────
		r.Group(func(r chi.Router) {
			r.Use(handlers.Authenticate(authSvc))

			r.Post("/auth/logout", authHandler.Logout)
			r.Post("/auth/logout-all", authHandler.LogoutAll)

			// ── Members ───────────────────────────────────────────────────
			r.Route("/members", func(r chi.Router) {
				r.With(handlers.RequireProfile("leadership")).Get("/", memberHandler.ListMembers)
				r.With(handlers.RequireProfile("leadership")).Post("/", memberHandler.CreateMember)
				r.With(handlers.RequireProfile("leadership")).Post("/import", memberHandler.ImportMembers)

				// /me routes must precede /{id} to avoid chi treating "me" as a param.
				r.Get("/me", memberHandler.GetMe)
				r.Put("/me", memberHandler.UpdateMe)
				r.Get("/me/instruments", memberHandler.GetMyInstruments)
				r.Post("/me/instruments", memberHandler.AddMyInstrument)
				r.Delete("/me/instruments/{instrument_id}", memberHandler.RemoveMyInstrument)

				r.With(handlers.RequireProfile("leadership")).Get("/{id}", memberHandler.GetMemberByID)
				r.With(handlers.RequireProfile("leadership")).Put("/{id}", memberHandler.UpdateMemberByID)
				r.With(handlers.RequireProfile("pastor")).Delete("/{id}", memberHandler.DeactivateMember)
				r.With(handlers.RequireProfile("leadership")).Get("/{id}/roles", memberHandler.GetMemberRoles)
				r.With(handlers.RequireProfile("leadership")).Post("/{id}/roles", memberHandler.AssignRole)
				r.With(handlers.RequireProfile("leadership")).Delete("/{id}/roles/{role_id}", memberHandler.RemoveMemberRole)

				r.Get("/{id}/instruments", memberHandler.GetMemberInstruments)
				r.With(handlers.RequireProfile("leadership")).Post("/{id}/instruments", memberHandler.AddMemberInstrument)
				r.With(handlers.RequireProfile("leadership")).Delete("/{id}/instruments/{instrument_id}", memberHandler.RemoveMemberInstrument)
			})

			// ── Roles ─────────────────────────────────────────────────────
			r.Route("/roles", func(r chi.Router) {
				r.With(handlers.RequireProfile("leadership")).Get("/", roleHandler.ListRoles)
				r.With(handlers.RequireProfile("pastor")).Post("/", roleHandler.CreateRole)
				r.With(handlers.RequireProfile("pastor")).Put("/{id}", roleHandler.UpdateRole)
				r.With(handlers.RequireProfile("pastor")).Delete("/{id}", roleHandler.DeleteRole)
			})

			// ── Instruments ───────────────────────────────────────────────
			r.Route("/instruments", func(r chi.Router) {
				r.Get("/", instrumentHandler.ListInstruments)
				r.With(handlers.RequireProfile("leadership")).Post("/", instrumentHandler.CreateInstrument)
				r.With(handlers.RequireProfile("leadership")).Delete("/{id}", instrumentHandler.DeleteInstrument)
			})

			// ── Inventory ─────────────────────────────────────────────────
			r.Route("/inventory", func(r chi.Router) {
				// Categories
				r.Route("/categories", func(r chi.Router) {
					r.Get("/", inventoryHandler.ListCategories)
					r.With(handlers.RequireProfile("leadership")).Post("/", inventoryHandler.CreateCategory)
					r.With(handlers.RequireProfile("leadership")).Put("/{id}", inventoryHandler.UpdateCategory)
					r.With(handlers.RequireProfile("leadership")).Delete("/{id}", inventoryHandler.DeleteCategory)
				})

				// Items
				r.Route("/items", func(r chi.Router) {
					r.Get("/", inventoryHandler.ListItems)
					r.With(handlers.RequireProfile("leadership")).Post("/", inventoryHandler.CreateItem)
					r.Get("/{id}", inventoryHandler.GetItemByID)
					r.With(handlers.RequireProfile("leadership")).Put("/{id}", inventoryHandler.UpdateItem)
					r.With(handlers.RequireProfile("leadership")).Post("/{id}/photo", inventoryHandler.UploadPhoto)
					r.With(handlers.RequireProfile("leadership")).Post("/{id}/discard", inventoryHandler.DiscardItem)
					r.With(handlers.RequireProfile("leadership")).Post("/{id}/donate", inventoryHandler.DonateItem)
				})

				// Loans
				r.Route("/loans", func(r chi.Router) {
					r.With(handlers.RequireProfile("leadership")).Get("/", inventoryHandler.ListLoans)
					r.Post("/", inventoryHandler.CreateLoan)
					r.With(handlers.RequireProfile("leadership")).Get("/{id}", inventoryHandler.GetLoanByID)
					r.With(handlers.RequireProfile("leadership")).Post("/{id}/approve", inventoryHandler.ApproveLoan)
					r.With(handlers.RequireProfile("leadership")).Post("/{id}/reject", inventoryHandler.RejectLoan)
					r.With(handlers.RequireProfile("leadership")).Post("/{id}/return", inventoryHandler.ReturnLoan)
				})
			})
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
