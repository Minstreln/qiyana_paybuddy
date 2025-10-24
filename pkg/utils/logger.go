package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/sirupsen/logrus"
)

var Logger = logrus.New()

func InitLogger() {
	env := os.Getenv("APP_ENV")

	Logger.SetReportCaller(true)

	Logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05Z07:00",
		PrettyPrint:     false,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := filepath.Base(f.File)
			return "", filename + ":" + strconv.Itoa(f.Line)
		},
	})

	level := os.Getenv("LOG_LEVEL")
	switch level {
	case "debug":
		Logger.SetLevel(logrus.DebugLevel)
	case "warn":
		Logger.SetLevel(logrus.WarnLevel)
	case "error":
		Logger.SetLevel(logrus.ErrorLevel)
	default:
		Logger.SetLevel(logrus.InfoLevel)
	}

	if env == "production" {
		_, currentFile, _, ok := runtime.Caller(0)
		if !ok {
			Logger.Out = os.Stdout
			Logger.Warn("Failed to get caller information, using stdout instead")
			return
		}
		projectRoot := filepath.Join(filepath.Dir(currentFile), "../..")
		var err error
		projectRoot, err = filepath.Abs(projectRoot)
		if err != nil {
			Logger.Out = os.Stdout
			Logger.WithError(err).Warn("Failed to resolve project root path, using stdout instead")
			return
		}

		logDir := filepath.Join(projectRoot, "logs")

		if err := os.MkdirAll(logDir, 0755); err != nil {
			Logger.Out = os.Stdout
			Logger.WithError(err).Warn("Failed to create logs directory, using stdout instead")
			return
		}

		logFilePath := filepath.Join(logDir, "app.log")

		file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			Logger.Out = os.Stdout
			Logger.WithError(err).Warn("Failed to log to file, using stdout instead")
			return
		}
		Logger.Out = file
	} else {
		Logger.Out = os.Stdout
	}
}
