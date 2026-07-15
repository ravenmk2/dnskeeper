package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ravenmk2/dnskeeper/internal/app"
	"github.com/ravenmk2/dnskeeper/internal/config"
	"github.com/ravenmk2/dnskeeper/internal/log"
	"github.com/sirupsen/logrus"
)

func main() {
	configPath := flag.String("config", "./config.toml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		logrus.Fatalf("load config: %v", err)
	}

	logger := log.Setup(cfg.Log.Level, os.Stdout)
	logger.Info("dnskeeper starting")
	logger.Infof("config loaded from %s, listen=%s, log_level=%s", *configPath, cfg.Server.Listen, cfg.Log.Level)

	application, err := app.New(cfg, logger)
	if err != nil {
		logger.Fatalf("init app: %v", err)
	}

	seedCtx, seedCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := application.SeedAdmin(seedCtx); err != nil {
		logger.WithError(err).Warn("seed admin user failed")
	}
	seedCancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Infof("dnskeeper started, listening on %s", cfg.Server.Listen)
		if err := application.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.WithError(err).Fatal("server error")
		}
	}()

	sig := <-sigCh
	logger.Infof("received signal %s, shutting down", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := application.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("shutdown error")
	}
	logger.Info("server stopped")
}
