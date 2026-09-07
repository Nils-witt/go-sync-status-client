// Package di wires the application's dependency graph using samber/do. It
// is the only place that knows about concrete adapter types.
package di

import (
	"go-sync-status-client/internal/adapter/repository/backuptool"
	"go-sync-status-client/internal/adapter/repository/multi"
	"go-sync-status-client/internal/adapter/tray"
	"go-sync-status-client/internal/infrastructure/config"
	"go-sync-status-client/internal/usecase"
	"log/slog"
	"time"

	"github.com/samber/do/v2"
)

// New builds the application's injector with every service registered.
// configPath is the config.json location to load; an empty string falls
// back to config.DefaultPath(). logger is registered as a value so any
// provider can pull it in.
func New(configPath string, logger *slog.Logger) *do.RootScope {
	injector := do.New()

	do.ProvideValue(injector, logger)

	do.Provide(injector, func(i do.Injector) (config.Config, error) {
		logger := do.MustInvoke[*slog.Logger](i)

		path := configPath
		if path == "" {
			var err error
			path, err = config.DefaultPath()
			if err != nil {
				return config.Config{}, err
			}
		}
		logger.Info("loading config", "path", path)
		return config.Load(path, logger)
	})

	do.Provide(injector, func(i do.Injector) (usecase.StatusRepository, error) {
		logger := do.MustInvoke[*slog.Logger](i)
		cfg := do.MustInvoke[config.Config](i)

		entries := make([]multi.Entry, 0, len(cfg.Servers))
		for _, sc := range cfg.Servers {
			opts := []backuptool.Option{backuptool.WithLogger(logger)}
			if sc.BearerToken != "" {
				opts = append(opts, backuptool.WithBearerToken(sc.BearerToken))
			}
			logger.Info("using backuptool repository", "server", sc.Name, "base_url", sc.BaseURL)
			entries = append(entries, multi.Entry{Name: sc.Name, Repo: backuptool.NewRepository(sc.BaseURL, opts...)})
		}
		return multi.NewRepository(logger, entries...), nil
	})

	do.Provide(injector, func(i do.Injector) (*usecase.StatusService, error) {
		repo := do.MustInvoke[usecase.StatusRepository](i)
		logger := do.MustInvoke[*slog.Logger](i)
		return usecase.NewStatusService(repo, logger), nil
	})

	do.Provide(injector, func(i do.Injector) (*tray.App, error) {
		service := do.MustInvoke[*usecase.StatusService](i)
		logger := do.MustInvoke[*slog.Logger](i)
		cfg := do.MustInvoke[config.Config](i)
		refreshInterval := time.Duration(cfg.RefreshIntervalSeconds) * time.Second
		return tray.NewApp(service, logger, refreshInterval), nil
	})

	return injector
}
