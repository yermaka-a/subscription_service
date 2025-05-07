package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"subpub/internal/app"
	"subpub/internal/config"
	"subpub/internal/logger"
	"syscall"
)

func main() {
	cfg := config.MustLoad()
	fmt.Print(cfg)

	log := logger.SetupLogger()

	app := app.New(log, cfg.GetPort())
	go app.GRPCSrv.MustRun()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	signal := <-stop
	log.Info("stopping application", slog.String("signal", signal.String()))
	app.GRPCSrv.Stop()
	log.Info("application is stopped")
}
