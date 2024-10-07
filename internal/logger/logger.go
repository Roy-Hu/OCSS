package logger

import (
	logger_util "github.com/free5gc/util/logger"
	"github.com/sirupsen/logrus"
)

var (
	Log       *logrus.Logger
	NfLog     *logrus.Entry
	MainLog   *logrus.Entry
	InitLog   *logrus.Entry
	CfgLog    *logrus.Entry
	CtxLog    *logrus.Entry
	UtilLog   *logrus.Entry
	ServerLog *logrus.Entry
)

func init() {
	fieldsOrder := []string{
		logger_util.FieldNF,
		logger_util.FieldCategory,
	}

	Log = logger_util.New(fieldsOrder)

	NfLog = Log.WithField(logger_util.FieldNF, "OCSS")
	MainLog = NfLog.WithField(logger_util.FieldCategory, "Main")
	InitLog = NfLog.WithField(logger_util.FieldCategory, "Init")
	CfgLog = NfLog.WithField(logger_util.FieldCategory, "CFG")
	CtxLog = NfLog.WithField(logger_util.FieldCategory, "CTX")
	UtilLog = NfLog.WithField(logger_util.FieldCategory, "Util")
	ServerLog = NfLog.WithField(logger_util.FieldCategory, "Server")

}
