// Command api runs the HTTP API server.
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/antoniopaya22/go-rest-template/v3/internal/app"
	"github.com/antoniopaya22/go-rest-template/v3/internal/config"
	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/logging"
	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/version"
)

func main() {
	showVersion := flag.Bool("version", false, "print build information and exit")
	healthcheck := flag.Bool("healthcheck", false, "probe the liveness endpoint of a running server and exit")
	printConfig := flag.String("print-config", "", "print the configuration reference (env or markdown) and exit")
	flag.Parse()

	switch {
	case *showVersion:
		info := version.Get()
		fmt.Printf("%s (commit %s, built %s, %s)\n", info.Version, info.Commit, info.Date, info.GoVersion)
		return
	case *printConfig == "env":
		fmt.Print(config.EnvExample())
		return
	case *printConfig == "markdown":
		fmt.Print(config.Markdown())
		return
	case *printConfig != "":
		fail(fmt.Errorf("unknown -print-config format %q", *printConfig))
	}

	cfg, err := config.Load()
	if err != nil {
		fail(err)
	}

	if *healthcheck {
		if err := probe(cfg); err != nil {
			fail(err)
		}
		return
	}

	if err := run(cfg); err != nil {
		fail(err)
	}
}

func run(cfg config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := logging.New(os.Stdout, cfg.Log.Level, cfg.Log.Format)
	build := version.Get()
	logger.Info("starting", "app", cfg.App.Name, "env", cfg.App.Environment, "version", build.Version, "commit", build.Commit)

	application, err := app.New(ctx, cfg, logger)
	if err != nil {
		return err
	}
	return application.Run(ctx)
}

// probe is the container health check: distroless images have no shell or
// curl, so the binary checks itself.
func probe(cfg config.Config) error {
	client := &http.Client{Timeout: 2 * time.Second}
	url := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(cfg.HTTP.Port)) + "/health/live"

	response, err := client.Get(url) //nolint:noctx // bounded by the client timeout.
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned %s", response.Status)
	}
	return nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
