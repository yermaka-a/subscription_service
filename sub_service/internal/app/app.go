package app

import (
	"log/slog"
	grpcapp "subpub/internal/app/grpc"
)

type App struct {
	GRPCSrv grpcapp.GrpcApp
}

func New(log *slog.Logger, grpcPort int) *App {
	grpcApp := grpcapp.New(log, grpcPort)

	return &App{
		GRPCSrv: grpcApp,
	}
}
