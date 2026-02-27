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

func (s *FlagService) ListFlags(req *flag.ListFlagsRequest, srv flag.FlagService_ListFlagsServer) error {
	ctx := srv.Context()
	flags, err := s.source.GetFlags(ctx)
	if err != nil {
		return err
	}
	for _, f := range flags {
		pbFlag, err := f.ToProto()
		if err != nil {
			return err
		}
		if err := srv.Send(&flag.ListFlagsResponse{
			Flag: pbFlag,
		}); err != nil {
			return err
		}
	}
	return nil
}
