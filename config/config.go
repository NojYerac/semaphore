package config

import (
	"github.com/nojyerac/go-lib/audit"
	"github.com/nojyerac/go-lib/auth"
	"github.com/nojyerac/go-lib/config"
	"github.com/nojyerac/go-lib/db"
	"github.com/nojyerac/go-lib/health"
	"github.com/nojyerac/go-lib/log"
	"github.com/nojyerac/go-lib/tracing"
	"github.com/nojyerac/go-lib/transport"
	"github.com/nojyerac/go-lib/transport/http"
	"github.com/nojyerac/go-lib/version"
)

var (
	LogConfig    *log.Configuration
	AuthConfig   *auth.Configuration
	AuditConfig  *audit.Configuration
	DBConfig     *db.Configuration
	TransConfig  *transport.Configuration
	TraceConfig  *tracing.Configuration
	HealthConfig *health.Configuration
	HTTPConfig   *http.Configuration
)

func InitAndValidate() error {
	loader := config.NewConfigLoader(version.GetVersion().Name)
	LogConfig = log.NewConfiguration()
	if err := loader.RegisterConfig(LogConfig); err != nil {
		return err
	}
	AuthConfig = auth.NewConfiguration()
	if err := loader.RegisterConfig(AuthConfig); err != nil {
		return err
	}
	AuditConfig = audit.NewConfiguration()
	if err := loader.RegisterConfig(AuditConfig); err != nil {
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
	TraceConfig = &tracing.Configuration{}
	if err := loader.RegisterConfig(TraceConfig); err != nil {
		return err
	}
	TransConfig = transport.NewConfiguration()
	if err := loader.RegisterConfig(TransConfig); err != nil {
		return err
	}
	return loader.InitAndValidate()
}
