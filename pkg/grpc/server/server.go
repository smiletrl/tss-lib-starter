package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/smiletrl/tss-lib-starter/pkg/constants"
	pb "github.com/smiletrl/tss-lib-starter/pkg/grpc"
	"github.com/smiletrl/tss-lib-starter/pkg/party"
	"google.golang.org/grpc"
)

// Register the rpc server for p2p service.
func RegisterServer(id string, party party.Party) error {
	port := fmt.Sprintf(":%s", constants.TestGrpcHost[id][1])

	log.Printf("grpc server listens at local port: %v", port)

	lis, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}
	var keep = keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second, // If a client pings more than once every 5 seconds, terminate the connection
		PermitWithoutStream: true,            // Allow pings even when there are no active streams
	}

	var kasp = keepalive.ServerParameters{
		MaxConnectionIdle:     15 * time.Second, // If a client is idle for 15 seconds, send a GOAWAY
		MaxConnectionAge:      30 * time.Second, // If any connection is alive for more than 30 seconds, send a GOAWAY
		MaxConnectionAgeGrace: 5 * time.Second,  // Allow 5 seconds for pending RPCs to complete before forcibly closing connections
		Time:                  5 * time.Second,  // Ping the client if it is idle for 5 seconds to ensure the connection is still active
		Timeout:               1 * time.Second,  // Wait 1 second for the ping ack before assuming the connection is dead
	}
	s := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(keep),
		grpc.KeepaliveParams(kasp),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_opentracing.StreamServerInterceptor(),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			//grpc_opentracing.UnaryServerInterceptor(grpc_opentracing.WithTracer(tracer)),
			grpc_opentracing.UnaryServerInterceptor(),
		)),
	)
	pb.RegisterP2PServer(s, &server{party: party})
	if err := s.Serve(lis); err != nil {
		return err
	}
	return nil
}

// server is rpc server for p2p
type server struct {
	pb.UnimplementedP2PServer
	party party.Party
}

func (s *server) OnReceiveMessage(ctx context.Context, msg *pb.Message) (*emptypb.Empty, error) {
	// update local party data
	if err := s.party.OnReceiveMessage(ctx, constants.MessageType(msg.GetType()), msg.GetFromPid(), msg.GetIsBroadcast(), msg.GetContent()); err != nil {
		log.Printf("error processing party on receive message: %v", err)
		return nil, fmt.Errorf("error processing party on receive message: %w", err)
	}
	return nil, nil
}
