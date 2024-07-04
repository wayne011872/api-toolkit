package apitool

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"
)

func AutoGinApiServer(cfg *Config) (*http.Server, error) {
	if cfg.errorHandler == nil {
		return nil, errors.New("missing error handler")
	}

	server := NewGinApiServer(cfg.GinMode, cfg.Service).
		SetServerErrorHandler(cfg.errorHandler)

	if cfg.authMid != nil {
		server = server.SetAuth(cfg.authMid)
	}
	server = server.Middles(cfg.getMiddles()...).
		AddAPIs(cfg.apis...).
		SetTrustedProxies(cfg.TrustedProxies)

	if len(cfg.proms) > 0 {
		server = server.SetPromhttp(cfg.proms...)
	}

	if cfg.Logger != nil {
		authMode := "release"
		if cfg.IsMockAuth {
			authMode = "mock"
		}
		cfg.Logger.Infof("run api at port: [%d], auth mode: [%s]",
			cfg.ApiPort, authMode)
	}
	return server.GetServer(cfg.ApiPort), nil
}

func AutoGinApiRun(ctx context.Context, cfg *Config) error {
	var apiWait sync.WaitGroup
	server, err := AutoGinApiServer(cfg)
	if err != nil {
		return err
	}
	const fiveSecods = 5 * time.Second
	apiWait.Add(1)
	go func(srv *http.Server) {

		for {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				cfg.Logger.Fatalf("listen: %s", err)
				time.Sleep(fiveSecods)
			} else if err == http.ErrServerClosed {
				apiWait.Done()
				return
			}
		}
	}(server)

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), fiveSecods)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		cfg.Logger.Fatalf("Server forced to shutdown: %v", err)
	}
	apiWait.Wait()
	return nil
}
