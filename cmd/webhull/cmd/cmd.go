package cmd

import (
	"flag"
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/layer87-labs/webhull/internal/app/server"
	"github.com/layer87-labs/webhull/internal/pkg/config"
)

// Version is the binary version. Set at build time via:
//
//	go build -ldflags="-X github.com/layer87-labs/webhull/cmd/webhull/cmd.Version=1.2.3"
//
// Defaults to "dev" when not set.
var Version = "dev"

// Run is the application entrypoint — config loading, wiring, startup.
func Run() error {
	// Flags
	configPath := flag.String("config", "deploy/config.yaml", "Path to operational configuration file")
	pagesPath := flag.String("pages", "", "Path to pages/site structure file (optional, for split config)")
	validate := flag.Bool("validate", false, "Validate configuration and exit without starting the server")
	version := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *version {
		fmt.Println("web-hull", Version)
		return nil
	}

	// Load configuration (split or monolithic)
	cfg, err := config.Load(*configPath, *pagesPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if *validate {
		fmt.Printf("config OK — site=%q languages=%v pages=%d\n",
			cfg.Site.Name, cfg.I18n.Languages, len(cfg.Pages))
		return nil
	}

	// Setup logger
	var logger *zap.Logger
	if cfg.Server.Environment == "production" {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	defer logger.Sync() //nolint:errcheck

	logger.Info(
		"starting web-hull",
		zap.String("version", Version),
		zap.String("site", cfg.Site.Name),
		zap.String("url", cfg.Site.BaseURL),
		zap.String("environment", cfg.Server.Environment),
		zap.Strings("languages", cfg.I18n.Languages),
		zap.Int("pages", len(cfg.Pages)),
	)

	// Create and start server
	srv, err := server.New(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	return srv.Start()
}

// Exit prints err to stderr and exits with code 1.
// Called from main to keep main() minimal.
func Exit(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
