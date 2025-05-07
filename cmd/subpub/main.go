package main

import (
	"fmt"
	"subpub/internal/app"
	"subpub/internal/config"
	"subpub/internal/logger"
)

func main() {
	cfg := config.MustLoad()
	fmt.Print(cfg)

	log := logger.SetupLogger()

	app := app.New(log, cfg.GetPort())
	app.GRPCSrv.MustRun()
}
