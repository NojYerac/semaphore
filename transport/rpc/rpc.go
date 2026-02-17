package rpc

import (
	"github.com/nojyerac/semaphore/data"
	"google.golang.org/grpc"
)

func RegisterServices(source *data.Source) func(*grpc.Server) {
	return func(s *grpc.Server) {
		_ = s
		_ = source
	}
}
