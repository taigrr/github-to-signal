package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/taigrr/signalcli"
)

const (
	bytesPerMiB         = 1024 * 1024
	watchdogInterval    = 30 * time.Second
	watchdogConsecutive = 3
	defaultDaemonHost   = "127.0.0.1"
	// minJavaHeapMB floors the derived -Xmx so tiny limits don't truncate to a
	// non-viable JVM heap (matches signalcli's own watchdog derivation).
	minJavaHeapMB = 32
)

// startManagedDaemon launches and supervises the signal-cli daemon in-process
// so its JVM heap can be capped and its RSS bounded by a watchdog (signal-cli
// leaks memory over time). It returns the daemon's base URL for the JSON-RPC
// client and a stop func that stops both the watchdog and the daemon.
//
// ctx (cancelled on SIGINT/SIGTERM) drives only the watchdog goroutine. The
// daemon process is deliberately NOT tied to ctx: exec.CommandContext would
// SIGKILL it the moment ctx is cancelled, skipping signal-cli's graceful
// shutdown (state flush). Instead the returned stop func calls daemon.Stop,
// which sends an interrupt and waits for a clean exit.
func startManagedDaemon(ctx context.Context, cfg Config) (baseURL string, stop func(), err error) {
	host, port, perr := daemonHostPort(cfg.SignalURL)
	if perr != nil {
		return "", nil, fmt.Errorf("parse signal_url for managed daemon: %w", perr)
	}

	// Cap the initial JVM heap too. signalcli only auto-derives JavaMaxHeapMB
	// inside Watch (applied on the next restart), so without this the first
	// process would launch uncapped until the watchdog first trips. Mirror the
	// library's derivation (3/4 of the limit, floored to a JVM-viable minimum).
	heapMB := cfg.SignalJavaMaxHeapMB
	if heapMB <= 0 && cfg.SignalMemoryLimitMB > 0 {
		heapMB = cfg.SignalMemoryLimitMB * 3 / 4
		heapMB = max(heapMB, minJavaHeapMB)
	}

	daemon := signalcli.NewDaemon(signalcli.DaemonConfig{
		CLIPath:       cfg.SignalCLIPath,
		Account:       cfg.SignalAccount,
		HTTPHost:      host,
		HTTPPort:      port,
		JavaMaxHeapMB: heapMB,
	})

	if err := daemon.Start(context.Background()); err != nil {
		return "", nil, fmt.Errorf("start signal-cli daemon: %w", err)
	}

	watchCtx, cancel := context.WithCancel(ctx)
	// watchDone is closed when the watchdog goroutine has fully returned, so
	// stop can wait for any in-flight Watch restart to finish before calling
	// daemon.Stop (avoiding a concurrent stop/restart race on the process).
	watchDone := make(chan struct{})
	if cfg.SignalMemoryLimitMB > 0 {
		limitBytes := uint64(cfg.SignalMemoryLimitMB) * bytesPerMiB
		go func() {
			defer close(watchDone)
			werr := daemon.Watch(watchCtx, signalcli.WatchConfig{
				MemoryLimit:     limitBytes,
				Interval:        watchdogInterval,
				ConsecutiveHits: watchdogConsecutive,
				OnRestart: func(rss uint64) {
					log.Printf("signal-cli watchdog: RSS %d MiB exceeded limit %d MiB, restarting daemon",
						rss/bytesPerMiB, cfg.SignalMemoryLimitMB)
				},
				OnError: func(err error) {
					log.Printf("signal-cli watchdog error: %v", err)
				},
			})
			if watchCtx.Err() == nil && werr != nil {
				log.Printf("signal-cli watchdog stopped: %v", werr)
			}
		}()
		log.Printf("signal-cli managed daemon: memory limit %d MiB, watchdog every %s",
			cfg.SignalMemoryLimitMB, watchdogInterval)
	} else {
		close(watchDone)
		log.Printf("signal-cli managed daemon started (no memory watchdog; set signal_memory_limit_mb to enable)")
	}

	stop = func() {
		cancel()
		<-watchDone
		if err := daemon.Stop(); err != nil {
			log.Printf("signal-cli daemon stop error: %v", err)
		}
	}
	return daemon.BaseURL(), stop, nil
}

// daemonHostPort extracts the host and port the managed daemon should bind to
// from the configured signal_url.
func daemonHostPort(signalURL string) (host string, port int, err error) {
	parsed, err := url.Parse(signalURL)
	if err != nil {
		return "", 0, err
	}
	host = parsed.Hostname()
	if host == "" {
		host = defaultDaemonHost
	}

	portStr := parsed.Port()
	if portStr == "" {
		return "", 0, fmt.Errorf("signal_url %q has no port", signalURL)
	}
	port, err = strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port in signal_url %q: %w", signalURL, err)
	}
	return host, port, nil
}
