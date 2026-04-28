package log

import (
	"encoding/json"
	"fmt"
	stdlog "log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

var (
	logger         = newLogger(zapcore.AddSync(os.Stderr))
	loggerMu       sync.RWMutex
	sugar          = logger.Sugar()
	jsonBufferPool = buffer.NewPool()
)

type jsonPrettyEncoder struct {
	zapcore.Encoder
	pretty     bool
	stackArray bool
}

func (e *jsonPrettyEncoder) Clone() zapcore.Encoder {
	return &jsonPrettyEncoder{
		Encoder:    e.Encoder.Clone(),
		pretty:     e.pretty,
		stackArray: e.stackArray,
	}
}

func (e *jsonPrettyEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buf, err := e.Encoder.EncodeEntry(ent, fields)
	if err != nil || (!e.pretty && !e.stackArray) {
		return buf, err
	}

	var obj map[string]any
	if err := json.Unmarshal(buf.Bytes(), &obj); err != nil {
		return buf, nil
	}

	if e.stackArray {
		if stack, ok := obj["stacktrace"].(string); ok && stack != "" {
			obj["stacktrace"] = strings.Split(stack, "\n")
		}
	}

	var out []byte
	if e.pretty {
		out, err = json.MarshalIndent(obj, "", "  ")
	} else {
		out, err = json.Marshal(obj)
	}
	if err != nil {
		return buf, nil
	}
	out = append(out, '\n')

	newBuf := jsonBufferPool.Get()
	newBuf.AppendBytes(out)
	buf.Free()
	return newBuf, nil
}

func currentLogPath() string {
	return filepath.Join(os.TempDir(), "sparkle-service.log")
}

func previousLogPath() string {
	return filepath.Join(os.TempDir(), "sparkle-service.previous.log")
}

func rotateLogFiles() error {
	currentPath := currentLogPath()
	previousPath := previousLogPath()

	if err := os.Remove(previousPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := os.Rename(currentPath, previousPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func InitLogging() (*os.File, error) {
	if err := rotateLogFiles(); err != nil {
		return nil, err
	}

	logFile, err := os.OpenFile(currentLogPath(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	os.Stdout = logFile
	os.Stderr = logFile

	setLogger(newLogger(zapcore.AddSync(logFile)))
	redirectStdLog()
	return logFile, nil
}

func newLogger(ws zapcore.WriteSyncer) *zap.Logger {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := &jsonPrettyEncoder{
		Encoder:    zapcore.NewJSONEncoder(encoderCfg),
		pretty:     true,
		stackArray: true,
	}
	return zap.New(zapcore.NewCore(encoder, ws, zapcore.InfoLevel), zap.AddCaller(), zap.AddCallerSkip(1))
}

func setLogger(newLog *zap.Logger) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	logger = newLog
	sugar = newLog.Sugar()
}

func L() *zap.Logger {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return logger
}

func S() *zap.SugaredLogger {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return sugar
}

func Sync() {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	if logger != nil {
		_ = logger.Sync()
	}
}

func redirectStdLog() {
	stdLogger, err := zap.NewStdLogAt(L().WithOptions(zap.AddCallerSkip(1)), zap.InfoLevel)
	if err != nil {
		return
	}
	stdlog.SetOutput(stdLogger.Writer())
	stdlog.SetFlags(0)
}

func Fatal(v ...any) {
	S().Fatal(fmt.Sprint(v...))
}

func Print(v ...any) {
	S().Info(fmt.Sprint(v...))
}

func Printf(format string, v ...any) {
	S().Infof(format, v...)
}

func Println(v ...any) {
	S().Info(strings.TrimSuffix(fmt.Sprintln(v...), "\n"))
}
