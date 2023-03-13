package log

import (
	"OpenIM/pkg/common/config"
	"OpenIM/pkg/common/tracelog"
	"bufio"
	"context"

	//"bufio"
	"fmt"
	"os"
	"time"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
)

var logger *LogrusLogger
var ctxLogger *LogrusLogger

type LogrusLogger struct {
	*logrus.Logger
	Pid  int
	Type string
}

func init() {
	logger = loggerInit("")
	ctxLogger = ctxLoggerInit("")
}

func NewPrivateLog(moduleName string) {
	logger = loggerInit(moduleName)
}

func ctxLoggerInit(moduleName string) *LogrusLogger {
	var ctxLogger = logrus.New()
	ctxLogger.SetLevel(logrus.Level(config.Config.Log.RemainLogLevel))
	src, err := os.OpenFile(os.DevNull, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		panic(err.Error())
	}
	writer := bufio.NewWriter(src)
	ctxLogger.SetOutput(writer)
	ctxLogger.SetFormatter(&nested.Formatter{
		TimestampFormat: "2006-01-02 15:04:05.000",
		HideKeys:        false,
		FieldsOrder:     []string{"PID", "FilePath", "OperationID"},
	})
	if config.Config.Log.ElasticSearchSwitch {
		ctxLogger.AddHook(newEsHook(moduleName))
	}
	//Log file segmentation hook
	hook := NewLfsHook(time.Duration(config.Config.Log.RotationTime)*time.Hour, config.Config.Log.RemainRotationCount, moduleName)
	ctxLogger.AddHook(hook)
	return &LogrusLogger{
		ctxLogger,
		os.Getpid(),
		"ctxLogger",
	}
}

func loggerInit(moduleName string) *LogrusLogger {
	var logger = logrus.New()
	//All logs will be printed
	logger.SetLevel(logrus.Level(config.Config.Log.RemainLogLevel))
	//Close std console output
	//os.O_WRONLY | os.O_CREATE | os.O_APPEND
	src, err := os.OpenFile(os.DevNull, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		panic(err.Error())
	}
	writer := bufio.NewWriter(src)
	logger.SetOutput(writer)
	// logger.SetOutput(os.Stdout)
	//Log Console Print Style Setting

	logger.SetFormatter(&nested.Formatter{
		TimestampFormat: "2006-01-02 15:04:05.000",
		HideKeys:        false,
		FieldsOrder:     []string{"PID", "FilePath", "OperationID"},
	})
	logger.SetFormatter(&logrus.JSONFormatter{})
	//File name and line number display hook
	logger.AddHook(newFileHook())

	//Send logs to elasticsearch hook
	if config.Config.Log.ElasticSearchSwitch {
		logger.AddHook(newEsHook(moduleName))
	}
	//Log file segmentation hook
	hook := NewLfsHook(time.Duration(config.Config.Log.RotationTime)*time.Hour, config.Config.Log.RemainRotationCount, moduleName)
	logger.AddHook(hook)
	return &LogrusLogger{
		logger,
		os.Getpid(),
		"",
	}
}

func InfoKv(ctx context.Context, msg string, keysAndValues ...interface{}) {
	operationID := tracelog.GetOperationID(ctx)
	logger.WithFields(logrus.Fields{
		"OperationID": operationID,
		"PID":         logger.Pid,
		"Msg":         msg,
	}).Infoln(keysAndValues)
}

func DebugKv(ctx context.Context, msg string, keysAndValues ...interface{}) {

}

func ErrorKv(ctx context.Context, msg string, err error, keysAndValues ...interface{}) {

}

func WarnKv(ctx context.Context, msg string, err error, keysAndValues ...interface{}) {

}

func Info(OperationID string, args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"OperationID": OperationID,
		"PID":         logger.Pid,
	}).Infoln(args)
}

func Error(OperationID string, args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"OperationID": OperationID,
		"PID":         logger.Pid,
	}).Errorln(args)
}

func Debug(OperationID string, args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"OperationID": OperationID,
		"PID":         logger.Pid,
	}).Debugln(args)
}

//Deprecated
func Warning(token, OperationID, format string, args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"PID":         logger.Pid,
		"OperationID": OperationID,
	}).Warningf(format, args...)

}

//internal method
func argsHandle(OperationID string, fields logrus.Fields, args []interface{}) {
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			fields[fmt.Sprintf("%v", args[i])] = args[i+1]
		} else {
			fields[fmt.Sprintf("%v", args[i])] = ""
		}
	}
	fields["OperationID"] = OperationID
	fields["PID"] = logger.Pid
}
func NewInfo(OperationID string, args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"OperationID": OperationID,
		"PID":         logger.Pid,
	}).Infoln(args)
}
func NewError(OperationID string, args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"OperationID": OperationID,
		"PID":         logger.Pid,
	}).Errorln(args)
}
func NewDebug(OperationID string, args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"OperationID": OperationID,
		"PID":         logger.Pid,
	}).Debugln(args)
}
func NewWarn(OperationID string, args ...interface{}) {
	logger.WithFields(logrus.Fields{
		"OperationID": OperationID,
		"PID":         logger.Pid,
	}).Warnln(args)
}
