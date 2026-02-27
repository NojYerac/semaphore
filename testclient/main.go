package main

import (
	"context"
	"io"

	"github.com/nojyerac/go-lib/log"
	libgrpc "github.com/nojyerac/go-lib/transport/grpc"
	"github.com/nojyerac/semaphore/pb/flag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger := log.NewLogger(&log.Configuration{
		LogLevel:     "debug",
		HumanFrendly: true,
	}).WithField("service", "testclient")
	libgrpc.SetLogger(logger)
	ctx := log.WithLogger(context.Background(), logger)
	log.SetDefaultCtxLogger(logger)

	creds := insecure.NewCredentials()
	cc, err := libgrpc.ClientConn(
		"localhost:8080",
		libgrpc.WithDialOptions(grpc.WithTransportCredentials(creds)),
	)
	if err != nil {
		panic(err)
	}
	defer cc.Close()
	flagClient := flag.NewFlagServiceClient(cc)
	stream, err := flagClient.StreamFlags(ctx, &flag.StreamFlagsRequest{})
	if err != nil {
		panic(err)
	}
	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				logger.Info("flag stream closed by server")
				return
			}
			panic(err)
		}
		logger.Infof("Received flag: %s", resp.Flags.Name)
	}
}
