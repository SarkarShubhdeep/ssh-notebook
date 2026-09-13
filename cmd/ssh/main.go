// Command ssh runs the SSH notebook server.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SarkarShubhdeep/ssh-notebook/internal/server"
	"github.com/charmbracelet/ssh"
)

func main() {
	cfg := server.Config{
		Host:        env("SSH_HOST", "0.0.0.0"),
		Port:        env("SSH_PORT", "2222"),
		HostKeyPath: env("SSH_HOST_KEY", ".ssh/host_ed25519"),
		Password:    os.Getenv("NOTEBOOK_PASSWORD"),
		NotebookDir: env("NOTEBOOK_DIR", "notebook"),
	}

	srv, err := server.New(cfg)
	if err != nil {
		slog.Error("failed to create server", "err", err)
		os.Exit(1)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	slog.Info("starting ssh notebook",
		"addr", net(cfg.Host, cfg.Port),
		"notebook", cfg.NotebookDir,
		"auth", authMode(cfg.Password),
	)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			slog.Error("server error", "err", err)
			done <- syscall.SIGTERM
		}
	}()

	<-done
	slog.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		slog.Error("shutdown error", "err", err)
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func net(host, port string) string { return host + ":" + port }

func authMode(pw string) string {
	if pw == "" {
		return "open (no password)"
	}
	return "password"
}
