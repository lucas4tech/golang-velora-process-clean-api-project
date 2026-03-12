package logger

import (
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	instance *zap.Logger
	once     sync.Once
)

func Get() *zap.Logger {
	once.Do(func() {
		if testingMode() {
			instance = zap.NewNop()
			return
		}
		var cfg zap.Config
		if os.Getenv("APP_ENV") == "production" {
			cfg = zap.NewProductionConfig()
		} else {
			cfg = zap.NewDevelopmentConfig()
			cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
		var err error
		instance, err = cfg.Build()
		if err != nil {
			panic("failed to initialize logger: " + err.Error())
		}
	})
	return instance
}

func testingMode() bool {
	if os.Getenv("RANKMYAPP_SILENT_LOGS") == "1" {
		return true
	}

	name := os.Args[0]
	return strings.HasSuffix(name, ".test") || strings.Contains(name, "_test")
}

func Sync() {
	if instance != nil {
		_ = instance.Sync()
	}
}
