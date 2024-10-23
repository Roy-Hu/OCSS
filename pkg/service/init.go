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

var SYSTEM *SystemApp

var _ app.App = &SystemApp{}

type SystemApp struct {
	ocssCtx *ocss_context.OCSSContext
	cfg     *factory.Config

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	state     *ocss.Server
	processor *ocss.Processor
	rdc       *ocss.Rdc
}

func NewApp(ctx context.Context, cfg *factory.Config, tlsKeyLogPath string) (*SystemApp, error) {
	sys := &SystemApp{
		cfg: cfg,
		wg:  sync.WaitGroup{},
	}
	sys.SetLogEnable(cfg.GetLogEnable())
	sys.SetLogLevel(cfg.GetLogLevel())
	sys.SetReportCaller(cfg.GetLogReportCaller())
	ocss_context.Init()

	sys.ctx, sys.cancel = context.WithCancel(ctx)
	sys.ocssCtx = ocss_context.GetSelf()

	rdc, err := ocss.NewRdc(sys)
	if err != nil {
		return sys, err
	}
	sys.rdc = rdc

	processor, err := ocss.NewProcessor(sys)
	if err != nil {
		return sys, err
	}
	sys.processor = processor

	if sys.state, err = ocss.NewServer(sys); err != nil {
		return nil, err
	}

	SYSTEM = sys

	return sys, nil
}

func (a *SystemApp) CancelContext() context.Context {
	return a.ctx
}

func (a *SystemApp) Context() *ocss_context.OCSSContext {
	return a.ocssCtx
}

func (a *SystemApp) Config() *factory.Config {
	return a.cfg
}

func (c *SystemApp) SetLogEnable(enable bool) {
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

func (c *SystemApp) SetLogLevel(level string) {
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

func (c *SystemApp) SetReportCaller(reportCaller bool) {
	logger.MainLog.Infof("Report Caller is set to [%v]", reportCaller)
	if reportCaller == logger.Log.ReportCaller {
		return
	}
	c.Config().SetLogReportCaller(reportCaller)
	logger.Log.SetReportCaller(reportCaller)
}

func (a *SystemApp) Start() {
	logger.InitLog.Infoln("Server started")

	a.wg.Add(1)
	go a.listenShutdownEvent()

	if err := a.state.Run(context.Background(), &a.wg); err != nil {
		logger.MainLog.Fatalf("Run OCSS server failed: %+v", err)
	}
}

func (a *SystemApp) listenShutdownEvent() {
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

func (c *SystemApp) Terminate() {
	c.cancel()
}

func (c *SystemApp) terminateProcedure() {
	logger.MainLog.Infof("Terminating ocss...")
	c.CallServerStop()
}

func (a *SystemApp) CallServerStop() {
	if a.state != nil {
		a.state.Stop()
	}
}

func (a *SystemApp) Processor() *ocss.Processor {
	return a.processor
}

func (a *SystemApp) Rdc() *ocss.Rdc {
	return a.rdc
}
