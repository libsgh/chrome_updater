package main

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"os"
	"path/filepath"
)

var logger *zap.SugaredLogger

func InitLogger(level string) {
	writeSyncer := getLogWriter(level)
	encoder := getEncoder()
	core := zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel)
	l := zap.New(core, zap.AddCaller())
	logger = l.Sugar()
}
func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}
func getLogWriter(level string) zapcore.WriteSyncer {
	ws := io.MultiWriter(os.Stdout)
	if level == "1" {
		ex, err := os.Executable()
		if err != nil {
			logger.Panic(err)
		}
		exPath := filepath.Dir(ex)
		file, _ := os.Create(filepath.Join(exPath, "debug.log"))
		ws = io.MultiWriter(file, os.Stdout)
	}
	return zapcore.AddSync(ws)
}
