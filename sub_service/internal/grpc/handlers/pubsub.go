package handlers

import (
	"context"
	"fmt"
	"log/slog"
	pbv1 "subpub/proto/gen/go"
	"subpub/subpub"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type serverAPI struct {
	pbv1.UnimplementedPubSubServer
	eventBus subpub.SubPub
	log      *slog.Logger
}

func Register(gRPC *grpc.Server, log *slog.Logger) {
	eventBus := subpub.NewSubPub()
	pbv1.RegisterPubSubServer(gRPC, &serverAPI{eventBus: eventBus, log: log})
}

func (s *serverAPI) Subscribe(req *pbv1.SubscribeRequest, stream pbv1.PubSub_SubscribeServer) error {
	const op = "handlers.Subscribe"
	log := s.log.With(slog.String("op", op))
	if req.GetKey() == "" {
		return status.Error(codes.InvalidArgument, "key must not be empty")
	}

	log.With(slog.String("msg", fmt.Sprintf("received subscription request for key: %s", req.GetKey())))
	subscription, err := s.eventBus.Subscribe(req.GetKey(), func(msg interface{}) {
		if err := stream.Send(&pbv1.Event{Data: msg.(string)}); err != nil {
			log.Error("error sending message to stream", slog.String("msg", err.Error()))
		}
	})

	if err != nil {
		log.Error("failed to subscribe", slog.String("msg", err.Error()))
		return status.Errorf(codes.Internal, "failed to subscribe")
	}

	defer subscription.Unsubscribe()

	<-stream.Context().Done()
	return nil
}

func (s *serverAPI) Publish(ctx context.Context, req *pbv1.PublishRequest) (*emptypb.Empty, error) {
	const op = "handlers.Publish"
	log := s.log.With("op", op)
	if req.GetKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "key must not be empty")
	}
	err := s.eventBus.Publish(req.GetKey(), req.GetData())
	if err != nil {
		if err.Error() == "event isn't found" {
			return nil, status.Error(codes.NotFound, "no subscribers for this key")
		}
		log.Error("failed to publish", slog.String("msg", err.Error()))
		return nil, status.Error(codes.Internal, "failed to publish")
	}

	return &emptypb.Empty{}, nil
}
