package rpc

import (
	"context"

	"github.com/nojyerac/go-lib/tracing"
	"github.com/nojyerac/semaphore/data"
	"github.com/nojyerac/semaphore/pb/flag"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

func RegisterServices(source data.DataEngine) func(*grpc.Server) {
	return func(s *grpc.Server) {
		flag.RegisterFlagServiceServer(s, NewFlagService(source))
	}
}

func NewFlagService(source data.DataEngine) *FlagService {
	return &FlagService{
		source: source,
		t:      tracing.TracerForPackage(),
	}
}

type FlagService struct {
	flag.UnimplementedFlagServiceServer
	source data.DataEngine
	t      trace.Tracer
}

func (s *FlagService) ListFlags(req *flag.ListFlagsRequest, srv flag.FlagService_ListFlagsServer) error {
	ctx, span := s.t.Start(srv.Context(), "ListFlags")
	defer span.End()
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

func (s *FlagService) GetFlag(ctx context.Context, req *flag.GetFlagRequest) (*flag.GetFlagResponse, error) {
	ctx, span := s.t.Start(ctx, "GetFlag")
	defer span.End()
	f, err := s.source.GetFlagByID(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, nil
	}
	pbFlag, err := f.ToProto()
	if err != nil {
		return nil, err
	}
	return &flag.GetFlagResponse{
		Flag: pbFlag,
	}, nil
}

func (s *FlagService) CreateFlag(ctx context.Context, req *flag.CreateFlagRequest) (*flag.CreateFlagResponse, error) {
	ctx, span := s.t.Start(ctx, "CreateFlag")
	defer span.End()
	f, err := data.FeatureFlagFromProto(req.Flag)
	if err != nil {
		return nil, err
	}
	id, err := s.source.CreateFlag(ctx, f)
	if err != nil {
		return nil, err
	}
	return &flag.CreateFlagResponse{
		Id: id,
	}, nil
}

func (s *FlagService) UpdateFlag(ctx context.Context, req *flag.UpdateFlagRequest) (*flag.UpdateFlagResponse, error) {
	ctx, span := s.t.Start(ctx, "UpdateFlag")
	defer span.End()
	f, err := data.FeatureFlagFromProto(req.Flag)
	if err != nil {
		return nil, err
	}
	if err := s.source.UpdateFlag(ctx, f); err != nil {
		return nil, err
	}
	return &flag.UpdateFlagResponse{Success: true}, nil
}

func (s *FlagService) DeleteFlag(ctx context.Context, req *flag.DeleteFlagRequest) (*flag.DeleteFlagResponse, error) {
	ctx, span := s.t.Start(ctx, "DeleteFlag")
	defer span.End()
	if err := s.source.DeleteFlag(ctx, req.Id); err != nil {
		return nil, err
	}
	return &flag.DeleteFlagResponse{Success: true}, nil
}

func (s *FlagService) Evaluate(ctx context.Context, req *flag.EvaluateRequest) (*flag.EvaluateResponse, error) {
	ctx, span := s.t.Start(ctx, "Evaluate")
	defer span.End()
	enabled, err := s.source.EvaluateFlag(ctx, req.FlagId, req.UserId, req.GroupIds)
	if err != nil {
		return nil, err
	}
	return &flag.EvaluateResponse{
		Enabled: enabled,
	}, nil
}
