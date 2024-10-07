package service

import (
	"context"
	"io"
	"os"
	"runtime/debug"
	"sync"

	"github.com/sirupsen/logrus"

	ocss_context "github.com/comp590/ocss/internal/context"
	"github.com/comp590/ocss/internal/logger"
	"github.com/comp590/ocss/internal/ocss"
	"github.com/comp590/ocss/pkg/app"
	"github.com/comp590/ocss/pkg/factory"
)

var OCSS *OCSSApp

var _ app.App = &OCSSApp{}

type OCSSApp struct {
	ocssCtx *ocss_context.OCSSContext
	cfg     *factory.Config

	ocssServer *ocss.Server

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewApp(ctx context.Context, cfg *factory.Config, tlsKeyLogPath string) (*OCSSApp, error) {
	var err error

	ocssApp := &OCSSApp{
		cfg: cfg,
		wg:  sync.WaitGroup{},
	}
	ocssApp.SetLogEnable(cfg.GetLogEnable())
	ocssApp.SetLogLevel(cfg.GetLogLevel())
	ocssApp.SetReportCaller(cfg.GetLogReportCaller())
	ocss_context.Init(cfg.Configuration)

	ocssApp.ctx, ocssApp.cancel = context.WithCancel(ctx)
	ocssApp.ocssCtx = ocss_context.GetSelf()

	if ocssApp.ocssServer, err = ocss.NewServer(ocssApp); err != nil {
		return nil, err
	}

	OCSS = ocssApp

	return ocssApp, nil
}

func (a *OCSSApp) CancelContext() context.Context {
	return a.ctx
}

func (a *OCSSApp) Context() *ocss_context.OCSSContext {
	return a.ocssCtx
}

func (a *OCSSApp) Config() *factory.Config {
	return a.cfg
}

func (c *OCSSApp) SetLogEnable(enable bool) {
	logger.MainLog.Infof("Log enable is set to [%v]", enable)
	if enable && logger.Log.Out == os.Stderr {
		return
	} else if !enable && logger.Log.Out == io.Discard {
		return
	}

	c.Config().SetLogEnable(enable)
	if enable {
		logger.Log.SetOutput(os.Stderr)
	} else {
		logger.Log.SetOutput(io.Discard)

	}
}

func (c *OCSSApp) SetLogLevel(level string) {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		logger.MainLog.Warnf("Log level [%s] is invalid", level)
		return
	}

	logger.MainLog.Infof("Log level is set to [%s]", level)
	if lvl == logger.Log.GetLevel() {
		return
	}

	c.Config().SetLogLevel(level)
	logger.Log.SetLevel(lvl)
}

func (c *OCSSApp) SetReportCaller(reportCaller bool) {
	logger.MainLog.Infof("Report Caller is set to [%v]", reportCaller)
	if reportCaller == logger.Log.ReportCaller {
		return
	}
	c.Config().SetLogReportCaller(reportCaller)
	logger.Log.SetReportCaller(reportCaller)
}

func (a *OCSSApp) Start() {
	logger.InitLog.Infoln("Server started")

	a.wg.Add(1)
	go a.listenShutdownEvent()

	if err := a.ocssServer.Run(context.Background(), &a.wg); err != nil {
		logger.MainLog.Fatalf("Run OCSS server failed: %+v", err)
	}
}

func (a *OCSSApp) listenShutdownEvent() {
	defer func() {
		if p := recover(); p != nil {
			// Print stack for panic to log. Fatalf() will let program exit.
			logger.MainLog.Fatalf("panic: %v\n%s", p, string(debug.Stack()))
		}
		a.wg.Done()
	}()

	<-a.ctx.Done()
	a.terminateProcedure()
}

func (c *OCSSApp) Terminate() {
	c.cancel()
}

func (c *OCSSApp) terminateProcedure() {
	logger.MainLog.Infof("Terminating ocss...")
	c.CallServerStop()
}

func (a *OCSSApp) CallServerStop() {
	if a.ocssServer != nil {
		a.ocssServer.Stop()
	}
}
