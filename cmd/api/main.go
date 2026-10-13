// Command api starts the HTTP API server.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/antoniopaya22/go-rest-template/internal/app"
	"github.com/antoniopaya22/go-rest-template/internal/pkg/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

	application, err := app.New(ctx, cfg)
	if err != nil {
		log.Fatalf("create application: %v", err)
	}

	if err := application.Run(ctx); err != nil {
		log.Fatalf("run application: %v", err)
	}
}
