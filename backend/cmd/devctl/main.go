package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"devctl/internal/app"
	"devctl/internal/applog"
)

func main() {
	configPath := flag.String("config", "devctl.yaml", "path to workspace config file")
	addr := flag.String("addr", "127.0.0.1:4312", "address to listen on")
	flag.Parse()

	a, err := app.New(*configPath)
	if err != nil {
		log.Fatalf("startup failed: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:    *addr,
		Handler: a.Router(),
	}

	go func() {
		applog.Info("main", "devctl listening on http://%s", *addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	applog.Info("main", "shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.ShutdownTimeout())
	defer cancel()

	a.Shutdown(shutdownCtx)
	if err := srv.Shutdown(shutdownCtx); err != nil {
		applog.Error("main", "http shutdown error: %v", err)
	}
}
