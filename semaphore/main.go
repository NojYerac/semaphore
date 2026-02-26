package main

import (
	"context"
	"os"
	"os/signal"
	"sync"

	libdb "github.com/nojyerac/go-lib/db"
	"github.com/nojyerac/go-lib/health"
	"github.com/nojyerac/go-lib/log"
	"github.com/nojyerac/go-lib/metrics"
	"github.com/nojyerac/go-lib/tracing"
	"github.com/nojyerac/go-lib/transport"
	"github.com/nojyerac/go-lib/transport/grpc"
	libhttp "github.com/nojyerac/go-lib/transport/http"
	"github.com/nojyerac/go-lib/version"

	"github.com/nojyerac/semaphore/config"
	"github.com/nojyerac/semaphore/data/db"
	"github.com/nojyerac/semaphore/transport/http"
	"github.com/nojyerac/semaphore/transport/rpc"
)

func main() {
	version.SetSemVer("0.0.0")
	version.SetServiceName("semaphore")
	v := version.GetVersion()

	if err := config.InitAndValidate(); err != nil {
		panic(err)
	}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())

	logger := log.NewLogger(config.LogConfig).WithField("service", v.Name).Logger
	ctx = log.WithLogger(ctx, logger)

	tp := tracing.NewTracerProvider(config.TraceConfig)
	tracing.SetGlobal(tp)

	mp, metricHandler, err := metrics.NewMetricProvider()
	if err != nil {
		logger.WithError(err).Panic("failed to create meter provider")
	}
	metrics.SetGlobal(mp)

	checker := health.NewChecker(config.HealthConfig)

	database := libdb.NewDatabase(config.DBConfig, libdb.WithLogger(logger), libdb.WithHealthChecker(checker))
	if err := database.Open(ctx); err != nil {
		logger.WithError(err).Panic("failed to connect to database")
	}

	source, err := db.NewDataSource(ctx, database)
	if err != nil {
		logger.WithError(err).Panic("failed to create data source")
	}

	httpServer := libhttp.NewServer(
		config.HTTPConfig,
		libhttp.WithLogger(logger),
		libhttp.WithHealthChecker(checker),
		libhttp.WithMetricsHandler(metricHandler),
	)
	http.RegisterRoutes(source, httpServer)

	grpc.SetLogger(logger)
	grpcServer := grpc.NewServer(rpc.RegisterServices(source))

	trans, err := transport.NewTLSServer(
		config.TransConfig,
		transport.WithHTTP(httpServer),
		transport.WithGRPC(grpcServer),
	)
	if err != nil {
		logger.WithError(err).Panic("failed to initialize transport")
	}

	wg := new(sync.WaitGroup)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := checker.Start(ctx); err != nil && err != context.Canceled {
			logger.WithError(err).Panic("health checker failed")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := trans.Start(ctx); err != nil {
			logger.WithError(err).Panic("transport failed")
		}
	}()

	logger.Info("starting")
	<-sigChan
	cancel()
	logger.Info("stopping")
	wg.Wait()
	logger.Info("stopped")
}
