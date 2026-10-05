package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/company/service-registry/internal/auth"
	"github.com/company/service-registry/internal/config"
	"github.com/company/service-registry/internal/server"
	"github.com/company/service-registry/internal/storage"
	"github.com/company/service-registry/internal/telemetry"
	"github.com/company/service-registry/internal/webstatic"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"
	commit  = "dev"
	date    = time.Now().Format("2006-01-02")
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "registry",
		Short: "Service Registry - Lightweight service registry and health management",
		Long:  `A modern, self-hosted service registry and service health administration platform.`,
		RunE:  runServer,
	}

	// Global flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "Config file path")
	rootCmd.PersistentFlags().StringP("log-level", "l", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolP("dev", "d", false, "Development mode")

	// Server command
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Start the registry server",
		RunE:  runServer,
	}
	rootCmd.AddCommand(serverCmd)

	// Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("registry version %s (commit: %s, date: %s)\n", version, commit, date)
		},
	}
	rootCmd.AddCommand(versionCmd)

	// Migrate command (placeholder for now)
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Database migrations",
	}
	migrateUpCmd := &cobra.Command{
		Use:   "up",
		Short: "Run migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Info("Running migrations...")
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			db, err := storage.NewDatabase(context.Background(), cfg.Storage.Path)
			if err != nil {
				return err
			}
			defer db.Close()
			err = storage.RunMigrations(context.Background(), db)
			if err != nil {
				return err
			}
			slog.Info("Migrations completed")
			return nil
		},
	}
	migrateCmd.AddCommand(migrateUpCmd)
	rootCmd.AddCommand(migrateCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runServer(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	if configPath != "" {
		_ = os.Setenv("REGISTRY_CONFIG", configPath)
	}

	devMode, _ := cmd.Flags().GetBool("dev")
	if devMode {
		_ = os.Setenv("REGISTRY_DEV_MODE", "true")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", slog.Any("error", err))
		return err
	}

	if err := os.MkdirAll(filepath.Dir(cfg.Storage.Path), 0o755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Initialize telemetry
	tp, err := telemetry.NewTracerProvider(cfg.Telemetry)
	if err != nil {
		slog.Error("Failed to initialize tracer", slog.Any("error", err))
		return err
	}
	defer tp.Shutdown(context.Background())

	// Initialize database
	db, err := storage.NewDatabase(context.Background(), cfg.Storage.Path)
	if err != nil {
		slog.Error("Failed to initialize database", slog.Any("error", err))
		return err
	}
	defer db.Close()

	// Run migrations
	if err := storage.RunMigrations(context.Background(), db); err != nil {
		slog.Error("Failed to run migrations", slog.Any("error", err))
		return err
	}
	if cfg.Auth.Enabled {
		result, err := auth.NewRepository(db).Bootstrap(context.Background(), cfg.Auth.BootstrapAdminUsername, cfg.Auth.BootstrapAdminEmail, cfg.Auth.BootstrapCredentialPath)
		if err != nil {
			return fmt.Errorf("security bootstrap failed closed: %w", err)
		}
		if result.Created {
			slog.Info("Alauda first-time initialization completed", slog.String("username", result.Username), slog.String("credentialPath", result.CredentialPath))
		}
	}

	// Create server
	srv := server.New(db, cfg, webstatic.FileSystem())

	// Start server
	errChan := make(chan error, 1)
	go func() {
		slog.Info("Starting registry server",
			slog.String("address", cfg.Server.Address),
			slog.Int("port", cfg.Server.Port),
		)
		errChan <- srv.Start()
	}()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		if err != nil {
			slog.Error("Server error", slog.Any("error", err))
			return err
		}
	case sig := <-sigChan:
		slog.Info("Received signal", slog.String("signal", sig.String()))
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("Shutdown error", slog.Any("error", err))
			return err
		}
	}

	return nil
}
