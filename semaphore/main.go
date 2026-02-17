package main

import (
	"context"
	"os"
	"os/signal"
	"sync"

	libdb "github.com/nojyerac/go-lib/pkg/db"
	"github.com/nojyerac/go-lib/pkg/health"
	"github.com/nojyerac/go-lib/pkg/log"
	"github.com/nojyerac/go-lib/pkg/tracing"
	"github.com/nojyerac/go-lib/pkg/transport"
	"github.com/nojyerac/go-lib/pkg/transport/grpc"
	libhttp "github.com/nojyerac/go-lib/pkg/transport/http"
	"github.com/nojyerac/go-lib/pkg/version"

	"github.com/nojyerac/semaphore/config"
	"github.com/nojyerac/semaphore/data"
	"github.com/nojyerac/semaphore/data/db"
	"github.com/nojyerac/semaphore/transport/http"
	"github.com/nojyerac/semaphore/transport/rpc"
)

func main() {
	v := version.GetVersion()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())

	logger := log.NewLogger(config.LogConfig).With().Str("service", v.Name).Logger()
	ctx = logger.WithContext(ctx)

	tracer := tracing.NewTracerProvider(config.TraceConfig)
	tracing.SetGlobal(tracer)

	checker := health.NewChecker(config.HealthConfig)

	database := libdb.NewDatabase(config.DBConfig, libdb.WithLogger(&logger))
	if err := database.Open(ctx); err != nil {
		logger.Panic().Err(err).Msg("failed to connect to database")
	}

	source := data.NewSource(db.New(database))

	httpServer := libhttp.NewServer(
		config.HTTPConfig,
		libhttp.WithLogger(&logger),
		libhttp.WithHealthCheck(checker),
	)
	http.RegisterRoutes(source, httpServer.APIRoutes())

	if err := grpc.SetGrpcLogger(&logger); err != nil {
		logger.Panic().Err(err).Msg("failed to set gRPC logger")
	}
	grpcServer := grpc.NewServer(rpc.RegisterServices(source))

	trans, err := transport.NewTLSServer(
		config.TransConfig,
		transport.WithHTTP(httpServer),
		transport.WithGRPC(grpcServer),
	)
	if err != nil {
		logger.Panic().Err(err).Msg("failed to initialize transport")
	}

	wg := new(sync.WaitGroup)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := checker.Start(ctx); err != nil && err != context.Canceled {
			logger.Panic().Err(err).Msg("health checker failed")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := trans.Start(ctx); err != nil {
			logger.Panic().Err(err).Msg("transport failed")
		}
	}()

	logger.Info().Msg("starting")
	<-sigChan
	cancel()
	logger.Info().Msg("stopping")
	wg.Wait()
	logger.Info().Msg("stopped")
}
