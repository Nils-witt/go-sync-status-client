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

// backuptoolServer pairs a configured server's name with the
// backuptool.Repository that talks to it. backuptool.Repository satisfies
// usecase.StatusRepository, usecase.ReceiverRepository,
// usecase.ConnectionRepository and usecase.ChangeWatcher, so sources and receivers share the same per-server
// HTTP client and live status connection instead of each constructing
// their own.
type backuptoolServer struct {
	name string
	repo *backuptool.Repository
}

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

	do.Provide(injector, func(i do.Injector) ([]backuptoolServer, error) {
		logger := do.MustInvoke[*slog.Logger](i)
		cfg := do.MustInvoke[config.Config](i)

		servers := make([]backuptoolServer, 0, len(cfg.Servers))
		for _, sc := range cfg.Servers {
			opts := []backuptool.Option{backuptool.WithLogger(logger)}
			if sc.BearerToken != "" {
				opts = append(opts, backuptool.WithBearerToken(sc.BearerToken))
			}
			logger.Info("using backuptool repository", "server", sc.Name, "base_url", sc.BaseURL)
			servers = append(servers, backuptoolServer{name: sc.Name, repo: backuptool.NewRepository(sc.BaseURL, opts...)})
		}
		return servers, nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.StatusRepository, error) {
		logger := do.MustInvoke[*slog.Logger](i)
		servers := do.MustInvoke[[]backuptoolServer](i)

		entries := make([]multi.Entry, 0, len(servers))
		for _, s := range servers {
			entries = append(entries, multi.Entry{Name: s.name, Repo: s.repo})
		}
		return multi.NewRepository(logger, entries...), nil
	})

	do.Provide(injector, func(i do.Injector) (*usecase.StatusService, error) {
		repo := do.MustInvoke[usecase.StatusRepository](i)
		logger := do.MustInvoke[*slog.Logger](i)
		return usecase.NewStatusService(repo, logger), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.ReceiverRepository, error) {
		logger := do.MustInvoke[*slog.Logger](i)
		servers := do.MustInvoke[[]backuptoolServer](i)

		entries := make([]multi.ReceiverEntry, 0, len(servers))
		for _, s := range servers {
			entries = append(entries, multi.ReceiverEntry{Name: s.name, Repo: s.repo})
		}
		return multi.NewReceiverRepository(logger, entries...), nil
	})

	do.Provide(injector, func(i do.Injector) (*usecase.ReceiverService, error) {
		repo := do.MustInvoke[usecase.ReceiverRepository](i)
		logger := do.MustInvoke[*slog.Logger](i)
		return usecase.NewReceiverService(repo, logger), nil
	})

	do.Provide(injector, func(i do.Injector) (*usecase.ConnectionService, error) {
		servers := do.MustInvoke[[]backuptoolServer](i)
		entries := make([]multi.ConnectionEntry, 0, len(servers))
		for _, s := range servers {
			entries = append(entries, multi.ConnectionEntry{Name: s.name, Repo: s.repo})
		}
		return usecase.NewConnectionService(multi.NewConnectionRepository(entries...)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.ChangeWatcher, error) {
		servers := do.MustInvoke[[]backuptoolServer](i)
		watchers := make([]usecase.ChangeWatcher, 0, len(servers))
		for _, s := range servers {
			watchers = append(watchers, s.repo)
		}
		return multi.NewWatcher(watchers...), nil
	})

	do.Provide(injector, func(i do.Injector) (*tray.App, error) {
		service := do.MustInvoke[*usecase.StatusService](i)
		receiverService := do.MustInvoke[*usecase.ReceiverService](i)
		connectionService := do.MustInvoke[*usecase.ConnectionService](i)
		watcher := do.MustInvoke[usecase.ChangeWatcher](i)
		logger := do.MustInvoke[*slog.Logger](i)
		cfg := do.MustInvoke[config.Config](i)
		refreshInterval := time.Duration(cfg.RefreshIntervalSeconds) * time.Second
		return tray.NewApp(service, receiverService, connectionService, watcher, logger, refreshInterval), nil
	})

	return injector
}
