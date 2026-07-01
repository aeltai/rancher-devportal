package main

import (
	"os"
	"time"
)

type config struct {
	WatchNamespace    string
	FleetNamespace    string
	DefaultGitBranch  string
	DefaultGitSecret  string
	ReconcileInterval time.Duration
}

func loadConfig() config {
	interval := 15 * time.Second
	if v := os.Getenv("RECONCILE_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			interval = d
		}
	}
	ns := envOr("PLATFORM_NAMESPACE", "devportal-system")
	return config{
		WatchNamespace:    ns,
		FleetNamespace:    envOr("PLATFORM_FLEET_NAMESPACE", "fleet-default"),
		DefaultGitBranch:  envOr("PLATFORM_GIT_BRANCH", "main"),
		DefaultGitSecret:  envOr("PLATFORM_GIT_SECRET", "platform-git-credentials"),
		ReconcileInterval: interval,
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
