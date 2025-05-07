package grpcapp

import (
	"fmt"
	"log/slog"
	"net"
	pubsubgrpc "subpub/internal/grpc/handlers"

	"google.golang.org/grpc"
)

type GrpcApp interface {
	MustRun()
	Stop()
}

type app struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	port       int
}

func New(log *slog.Logger, port int) GrpcApp {
	gRPCServer := grpc.NewServer()

	pubsubgrpc.Register(gRPCServer, log)
	return &app{
		log:        log,
		gRPCServer: gRPCServer,
		port:       port,
	}
}

func (a *app) MustRun() {
	if err := a.run(); err != nil {
		panic(err)
	}
}

func (a *app) run() error {
	const op = "grpcapp.Run"
	log := a.log.With(slog.String("op", op), slog.Int("port", a.port))

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("grpc server is running", slog.String("addr", l.Addr().String()))

	if err := a.gRPCServer.Serve(l); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *app) Stop() {
	const op = "grpcapp.Stop"
	a.log.With(slog.String("op", op)).Info("stopping gRPC server", slog.Int("port", a.port))
	a.gRPCServer.GracefulStop()
}
