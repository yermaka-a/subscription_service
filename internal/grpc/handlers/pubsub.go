package handlers

import (
	"context"
	pbv1 "subpub/proto/gen/go"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type serverAPI struct {
	pbv1.UnimplementedPubSubServer
}

func Register(gRPC *grpc.Server) {
	pbv1.RegisterPubSubServer(gRPC, &serverAPI{})
}

func (s *serverAPI) Subscribe(req *pbv1.SubscribeRequest, stream pbv1.PubSub_SubscribeServer) error {
	return nil
}

func (s *serverAPI) Publish(ctx context.Context, req *pbv1.PublishRequest) (*emptypb.Empty, error) {
	return nil, nil
}
