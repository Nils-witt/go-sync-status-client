// Command go-sync-status-client runs a system tray application that
// displays sync status for a set of sources.
package main

import (
	"flag"
	"fmt"
	"go-sync-status-client/internal/adapter/tray"
	"go-sync-status-client/internal/infrastructure/di"
	"log/slog"
	"os"

	"github.com/samber/do/v2"
)

// version and commit are set via -ldflags at build time (see .goreleaser.yml).
var (
	version = "dev"
	commit  = "none"
)

func main() {
	configPath := flag.String("config", "", "path to config.json (default: $XDG_CONFIG_HOME/go-sync-status-client/config.json)")
	logLevel := flag.String("log-level", "info", "log level: debug, info, warn, error")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("go-sync-status-client %s (%s)\n", version, commit)
		return
	}

	logger := newLogger(*logLevel)

	injector := di.New(*configPath, logger)
	defer func() {
		if err := injector.Shutdown(); err != nil {
			logger.Error("shutdown failed", "error", err)
		}
	}()

	app := do.MustInvoke[*tray.App](injector)
	logger.Info("starting go-sync-status-client")
	app.Run()
}

// newLogger builds the process-wide logger. Invalid levels fall back to
// info rather than failing startup.
func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))
}
