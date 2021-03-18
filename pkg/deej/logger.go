package deej

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/omriharel/deej/pkg/deej/util"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	buildTypeNone    = ""
	buildTypeDev     = "dev"
	buildTypeRelease = "release"

	logDirectory = "logs"
	logFilename  = "deej-latest-run.log"
)

// NewLogger provides a logger instance for the whole program
func NewLogger(buildType string) (*zap.SugaredLogger, error) {
	var loggerConfig zap.Config

	// release: info and above, log to file only (no UI)
	if buildType == buildTypeRelease {
		if err := util.EnsureDirExists(logDirectory); err != nil {
			return nil, fmt.Errorf("ensure log directory exists: %w", err)
		}

		loggerConfig = zap.NewProductionConfig()

		loggerConfig.OutputPaths = []string{filepath.Join(logDirectory, logFilename)}
		loggerConfig.Encoding = "console"

		// development: debug and above, log to stderr only, colorful
	} else {
		loggerConfig = zap.NewDevelopmentConfig()

		// make it colorful
		loggerConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// all build types: make it readable
	loggerConfig.EncoderConfig.EncodeCaller = nil
	loggerConfig.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
	}

	loggerConfig.EncoderConfig.EncodeName = func(s string, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(fmt.Sprintf("%-27s", s))
	}

	logger, err := loggerConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("create zap logger: %w", err)
	}

	// no reason not to use the sugared logger - it's fast enough for anything we're gonna do
	sugar := logger.Sugar()

	return sugar, nil
}
