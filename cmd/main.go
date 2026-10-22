package main

import (
	in "ABIevm/internal"
	repository "ABIevm/internal/repository/stakedao"
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := runServer(); err != nil {
		slog.Error("server stopped", "reason", err.Error())
		os.Exit(1)
	}
}

func runServer() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var store *repository.RegistryPostgres
	if url := os.Getenv("DATABASE_URL"); url != "" {
		startup, cancel := context.WithTimeout(ctx, 10*time.Second)
		var err error
		store, err = repository.OpenRegistryPostgres(startup, url)
		cancel()
		if err != nil {
			return err
		} // Repository errors never include the credential-bearing DSN.
		defer store.Close()
		slog.Info("StakeDAO registry storage", "backend", "postgres")
	} else {
		slog.Info("StakeDAO registry storage", "backend", "disabled", "discovery", "full scan on each request")
	}
	var server *in.Server
	if store != nil {
		server = in.NewServerWithRegistry(store)
	} else {
		server = in.NewServer()
	}
	if err := server.Run(ctx); err != nil {
		return err
	}
	return nil
}
