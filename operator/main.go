package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := loadConfig()
	log.Printf("platform-operator starting namespace=%s fleetNs=%s interval=%s",
		cfg.WatchNamespace, cfg.FleetNamespace, cfg.ReconcileInterval)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	r, err := newReconciler(cfg)
	if err != nil {
		log.Fatalf("init reconciler: %v", err)
	}

	ticker := time.NewTicker(cfg.ReconcileInterval)
	defer ticker.Stop()

	// Run immediately, then on interval
	for {
		if err := r.reconcileAll(ctx); err != nil {
			log.Printf("reconcile error: %v", err)
		}
		select {
		case <-ctx.Done():
			log.Println("platform-operator stopped")
			return
		case <-ticker.C:
		}
	}
}
