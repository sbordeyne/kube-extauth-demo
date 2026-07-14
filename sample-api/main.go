package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sbordeyne/sample-api/pkg/auth"
	"github.com/sbordeyne/sample-api/pkg/config"
	"github.com/sbordeyne/sample-api/pkg/db"
	"github.com/sbordeyne/sample-api/pkg/openapi"
	"github.com/sbordeyne/sample-api/pkg/server"
)

const defaultConfigPath = "/etc/sample-api/config.yaml"

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	configPath := flag.String("config", envOr("CONFIG_FILE", defaultConfigPath), "path to config.yaml")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	// Apply pending migrations (bundled in the binary) before serving.
	if err := db.RunMigrations(cfg.Database.URL); err != nil {
		return err
	}
	log.Println("migrations applied")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		return err
	}
	defer pool.Close()

	privKey, err := auth.LoadPrivateKey(cfg.JWT.PrivateKeyFile)
	if err != nil {
		return err
	}
	signer := auth.NewSigner(privKey)

	api := openapi.NewServer(pool, signer)
	return server.Run(ctx, cfg.HTTP.Public, cfg.HTTP.Health, api.Handler(), server.HealthHandler(pool))
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
