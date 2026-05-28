package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"meridian-go-rewrite/internal/cli"
	"meridian-go-rewrite/internal/config"
	"meridian-go-rewrite/internal/logger"
)

func main() {
	if err := config.LoadEnv(".env"); err != nil {
		fmt.Printf("Note: .env load skipped or failed: %v\n", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		for sig := range sigCh {
			switch sig {
			case syscall.SIGHUP:
				if err := config.HotReload(); err != nil {
					logger.Error("sighup", err)
				} else {
					logger.Log("sighup", "config reloaded")
				}
			case syscall.SIGINT, syscall.SIGTERM:
				fmt.Printf("\nReceived %v. Shutting down...\n", sig)
				os.Exit(0)
			}
		}
	}()

	if err := cli.RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
