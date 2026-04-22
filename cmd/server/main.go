package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/carlos-loya/water-quality-data-management/internal/alerts"
	"github.com/carlos-loya/water-quality-data-management/internal/api"
	"github.com/carlos-loya/water-quality-data-management/internal/events"
	"github.com/carlos-loya/water-quality-data-management/internal/storage"
	"github.com/carlos-loya/water-quality-data-management/internal/storage/blob"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	dbURL := envOr("DATABASE_URL", "postgres://wqm:wqm_dev@localhost:5432/water_quality?sslmode=disable")
	migrationsPath := envOr("MIGRATIONS_PATH", "file://migrations")
	natsURL := envOr("NATS_URL", "nats://localhost:4222")
	jwtSecret := envOr("JWT_SECRET", "dev-secret-change-in-production")
	addr := envOr("ADDR", ":8080")
	alertInterval := parseDuration(envOr("ALERT_EVAL_INTERVAL", "5m"), 5*time.Minute)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	db, err := storage.Connect(ctx, dbURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := storage.Migrate(dbURL, migrationsPath); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	bus, err := events.Connect(natsURL)
	if err != nil {
		slog.Error("failed to connect to nats", "error", err)
		os.Exit(1)
	}
	defer bus.Close()

	// Start the audit log consumer — writes change events to the audit_log table.
	if _, err := events.NewAuditConsumer(db, bus); err != nil {
		slog.Error("failed to start audit consumer", "error", err)
		os.Exit(1)
	}

	queries := storage.New(db)

	// Start the alerts evaluator — scans exceedances + overdue calibrations on a ticker.
	evaluator := alerts.NewEvaluator(queries, bus, alertInterval)
	go evaluator.Run(ctx)

	blobs, err := newBlobStore(ctx)
	if err != nil {
		slog.Error("failed to init blob store", "error", err)
		os.Exit(1)
	}

	router := api.NewRouter(queries, bus, blobs, jwtSecret)

	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(s string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		slog.Warn("invalid duration, using fallback", "value", s, "fallback", fallback)
		return fallback
	}
	return d
}

func newBlobStore(ctx context.Context) (blob.Store, error) {
	backend := envOr("BLOB_BACKEND", "fs")
	switch backend {
	case "fs":
		dir := envOr("ATTACHMENT_DIR", "./data/attachments")
		slog.Info("blob store: filesystem", "dir", dir)
		return blob.NewFSStore(dir)
	case "s3":
		pathStyle, _ := strconv.ParseBool(envOr("S3_FORCE_PATH_STYLE", "true"))
		cfg := blob.S3Config{
			Endpoint:       os.Getenv("S3_ENDPOINT"),
			Region:         envOr("S3_REGION", "us-east-1"),
			Bucket:         os.Getenv("S3_BUCKET"),
			AccessKey:      os.Getenv("S3_ACCESS_KEY"),
			SecretKey:      os.Getenv("S3_SECRET_KEY"),
			ForcePathStyle: pathStyle,
		}
		slog.Info("blob store: s3", "endpoint", cfg.Endpoint, "bucket", cfg.Bucket)
		return blob.NewS3Store(ctx, cfg)
	default:
		return nil, fmt.Errorf("unknown BLOB_BACKEND %q", backend)
	}
}
