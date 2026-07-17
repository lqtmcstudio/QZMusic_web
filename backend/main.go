package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}
	store, err := openStore(cfg.DatabasePath, cfg.QuotaLocation)
	if err != nil {
		log.Fatal(err)
	}
	defer store.db.Close()

	auth := newAuthService(cfg, store)
	history := newHistoryService(cfg, store)
	workerContext, stopWorkers := context.WithCancel(context.Background())
	defer stopWorkers()
	history.Start(workerContext)
	app := &Server{cfg: cfg, store: store, auth: auth}
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           app.routes(),
		ReadHeaderTimeout: 8 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       75 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		log.Printf("QZ Music API listening on %s", cfg.HTTPAddr)
		if !auth.configured() {
			log.Printf("Re-Link SSO is not configured; public reading remains available")
		}
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	stopWorkers()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
