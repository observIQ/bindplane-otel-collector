//go:build !darwin

package logging

import (
	"fmt"
	"os"

	"go.uber.org/zap/zapcore"
)

func (l *LoggerConfig) core() (zapcore.Core, error) {
	cores := []zapcore.Core{}
	for _, ot := range l.outputTypes() {
		switch ot {
		case appleOutput:
			return nil, fmt.Errorf("apple logging is only available on darwin systems")
		case fileOutput:
			cores = append(cores, zapcore.NewCore(newEncoder(), zapcore.AddSync(l.File), l.Level))
		case stdOutput:
			cores = append(cores, zapcore.NewCore(newEncoder(), zapcore.Lock(os.Stdout), l.Level))
		default:
			return nil, fmt.Errorf("unrecognized output type: %s", ot)
		}
	}
	return zapcore.NewTee(cores...), nil
}
