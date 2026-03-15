package worker

import (
	"fmt"

	"github.com/a7medalyapany/GoBank.git/logger"
	"go.uber.org/zap"
)

// Logger adapts our zap-based logger to satisfy asynq's Logger interface.
// asynq expects: Debug, Info, Warn, Error, Fatal - all with variadic interface{} args.
type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) Debug(args ...interface{}) {
	logger.G().Debug(fmt.Sprint(args...), zap.String("source", "asynq"))
}

func (l *Logger) Info(args ...interface{}) {
	logger.G().Info(fmt.Sprint(args...), zap.String("source", "asynq"))
}

func (l *Logger) Warn(args ...interface{}) {
	logger.G().Warn(fmt.Sprint(args...), zap.String("source", "asynq"))
}

func (l *Logger) Error(args ...interface{}) {
	logger.G().Error(fmt.Sprint(args...), zap.String("source", "asynq"))
}

func (l *Logger) Fatal(args ...interface{}) {
	logger.G().Fatal(fmt.Sprint(args...), zap.String("source", "asynq"))
}