package app

import (
	ocss_context "github.com/comp590/ocss/internal/context"
	"github.com/comp590/ocss/pkg/factory"
)

type App interface {
	SetLogEnable(enable bool)
	SetLogLevel(level string)
	SetReportCaller(reportCaller bool)

	Start()
	Terminate()

	Context() *ocss_context.OCSSContext
	Config() *factory.Config
}
