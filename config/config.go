package config

import (
	"github.com/nojyerac/go-lib/pkg/config"
	"github.com/nojyerac/go-lib/pkg/db"
	"github.com/nojyerac/go-lib/pkg/health"
	"github.com/nojyerac/go-lib/pkg/log"
	"github.com/nojyerac/go-lib/pkg/tracing"
	"github.com/nojyerac/go-lib/pkg/transport"
	"github.com/nojyerac/go-lib/pkg/transport/http"
	"github.com/nojyerac/go-lib/pkg/version"
)

var (
	LogConfig    *log.Configuration
	DBConfig     *db.Configuration
	TraceConfig  *tracing.Configuration
	TransConfig  *transport.Configuration
	HealthConfig *health.Configuration
	HTTPConfig   *http.Configuration
)

func InitAndValidate() error {
	loader := config.NewConfigLoader(version.GetVersion().Name)
	LogConfig = log.NewConfiguration()
	if err := loader.RegisterConfig(LogConfig); err != nil {
		return err
	}
	DBConfig = db.NewConfiguration()
	if err := loader.RegisterConfig(DBConfig); err != nil {
		return err
	}
	HealthConfig = health.NewConfiguration()
	if err := loader.RegisterConfig(HealthConfig); err != nil {
		return err
	}
	HTTPConfig = http.NewConfiguration()
	if err := loader.RegisterConfig(HTTPConfig); err != nil {
		return err
	}
	TraceConfig = tracing.NewConfiguration()
	if err := loader.RegisterConfig(TraceConfig); err != nil {
		return err
	}
	TransConfig = transport.NewConfiguration()
	if err := loader.RegisterConfig(TransConfig); err != nil {
		return err
	}
	return loader.InitAndValidate()
}
