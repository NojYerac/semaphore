package rpc

import (
	"github.com/nojyerac/semaphore/data"
	"github.com/nojyerac/semaphore/pb/flag"
	"google.golang.org/grpc"
)

func RegisterServices(source data.Source) func(*grpc.Server) {
	return func(s *grpc.Server) {
		flag.RegisterFlagServiceServer(s, NewFlagService(source))
	}
}

func NewFlagService(source data.Source) *FlagService {
	return &FlagService{
		source: source,
	}
}

type FlagService struct {
	flag.UnimplementedFlagServiceServer
	source data.Source
}

func (s *FlagService) StreamFlags(req *flag.StreamFlagsRequest, srv flag.FlagService_StreamFlagsServer) error {
	ctx := srv.Context()
	flags, err := s.source.GetFlags(ctx)
	if err != nil {
		return err
	}
	for _, f := range flags {
		if err := srv.Send(&flag.StreamFlagsResponse{
			Flags: &flag.Flag{
				Id:   int64(f.ID),
				Name: f.Name,
			},
		}); err != nil {
			return err
		}
	}
	return nil
}
