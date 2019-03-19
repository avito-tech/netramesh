package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

var (
	logLock       sync.Mutex
	defaultLogger *Logger
)

type Level int

const (
	initText                  = "Logger Init wasn't called"
	flags                     = log.Ldate | log.Lmicroseconds | log.Lshortfile
	EnvNetraLoggerLevel       = "NETRA_LOGGER_LEVEL"
	FatalLevel          Level = iota
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
)

type Logger struct {
	outputLevel Level
	inner       *log.Logger
	closer      io.Closer
	initialized bool
}

func init() {
	defaultLogger = &Logger{
		outputLevel: DebugLevel,
		inner:       log.New(os.Stderr, initText, flags),
	}
}

func Init(name string, errorLevel string, logFile io.Writer) (*Logger, error) {
	level := InfoLevel
	if errorLevel != "" {
		errorLevel = strings.ToLower(errorLevel)
		switch errorLevel {
		case "fatal":
			level = FatalLevel
		case "error":
			level = ErrorLevel
		case "warning":
			level = WarnLevel
		case "warn":
			level = WarnLevel
		case "info":
			level = InfoLevel
		case "debug":
			level = DebugLevel
		default:
			return nil, fmt.Errorf("invalid logger level %s", errorLevel)
		}
	}
	var il io.Writer

	iLogs := []io.Writer{logFile}
	if il != nil {
		iLogs = append(iLogs, il)
	}

	l := Logger{
		outputLevel: level,
		inner:       log.New(io.MultiWriter(iLogs...), name, flags),
	}
	l.closer = logFile.(io.Closer)
	l.initialized = true

	logLock.Lock()
	defer logLock.Unlock()
	if !defaultLogger.initialized {
		defaultLogger = &l
	}

	return &l, nil
}

func (l *Logger) Close() {
	logLock.Lock()
	defer logLock.Unlock()
	if err := l.closer.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Error on log %v closing %v\n", l.closer, err)
	}
}

func (l *Logger) Debug(v ...interface{}) {
	l.output(DebugLevel, 0, fmt.Sprint(v...))
}

func (l *Logger) DebugDepth(depth int, v ...interface{}) {
	l.output(DebugLevel, depth, fmt.Sprint(v...))
}

func (l *Logger) Debugln(v ...interface{}) {
	l.output(DebugLevel, 0, fmt.Sprintln(v...))
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	l.output(DebugLevel, 0, fmt.Sprintf(format, v...))
}

func (l *Logger) Info(v ...interface{}) {
	l.output(InfoLevel, 0, fmt.Sprint(v...))
}

func (l *Logger) InfoDepth(depth int, v ...interface{}) {
	l.output(InfoLevel, depth, fmt.Sprint(v...))
}

func (l *Logger) Infoln(v ...interface{}) {
	l.output(InfoLevel, 0, fmt.Sprintln(v...))
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.output(InfoLevel, 0, fmt.Sprintf(format, v...))
}

func (l *Logger) Warning(v ...interface{}) {
	l.output(WarnLevel, 0, fmt.Sprint(v...))
}

func (l *Logger) WarningDepth(depth int, v ...interface{}) {
	l.output(WarnLevel, depth, fmt.Sprint(v...))
}

func (l *Logger) Warningln(v ...interface{}) {
	l.output(WarnLevel, 0, fmt.Sprintln(v...))
}

func (l *Logger) Warningf(format string, v ...interface{}) {
	l.output(WarnLevel, 0, fmt.Sprintf(format, v...))
}

func (l *Logger) Error(v ...interface{}) {
	l.output(ErrorLevel, 0, fmt.Sprint(v...))
}

func (l *Logger) ErrorDepth(depth int, v ...interface{}) {
	l.output(ErrorLevel, depth, fmt.Sprint(v...))
}

func (l *Logger) Errorln(v ...interface{}) {
	l.output(ErrorLevel, 0, fmt.Sprintln(v...))
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.output(ErrorLevel, 0, fmt.Sprintf(format, v...))
}

func (l *Logger) Fatal(v ...interface{}) {
	l.output(FatalLevel, 0, fmt.Sprint(v...))
	l.Close()
	os.Exit(1)
}

func (l *Logger) FatalDepth(depth int, v ...interface{}) {
	l.output(FatalLevel, depth, fmt.Sprint(v...))
	l.Close()
	os.Exit(1)
}

func (l *Logger) Fatalln(v ...interface{}) {
	l.output(FatalLevel, 0, fmt.Sprintln(v...))
	l.Close()
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.output(FatalLevel, 0, fmt.Sprintf(format, v...))
	l.Close()
	os.Exit(1)
}

func Info(v ...interface{}) {
	defaultLogger.output(InfoLevel, 0, fmt.Sprint(v...))
}

func InfoDepth(depth int, v ...interface{}) {
	defaultLogger.output(InfoLevel, depth, fmt.Sprint(v...))
}

func Infoln(v ...interface{}) {
	defaultLogger.output(InfoLevel, 0, fmt.Sprintln(v...))
}

func Infof(format string, v ...interface{}) {
	defaultLogger.output(InfoLevel, 0, fmt.Sprintf(format, v...))
}

func Warning(v ...interface{}) {
	defaultLogger.output(WarnLevel, 0, fmt.Sprint(v...))
}

func WarningDepth(depth int, v ...interface{}) {
	defaultLogger.output(WarnLevel, depth, fmt.Sprint(v...))
}

func Warningln(v ...interface{}) {
	defaultLogger.output(WarnLevel, 0, fmt.Sprintln(v...))
}

func Warningf(format string, v ...interface{}) {
	defaultLogger.output(WarnLevel, 0, fmt.Sprintf(format, v...))
}

func Error(v ...interface{}) {
	defaultLogger.output(ErrorLevel, 0, fmt.Sprint(v...))
}

func ErrorDepth(depth int, v ...interface{}) {
	defaultLogger.output(ErrorLevel, depth, fmt.Sprint(v...))
}

func Errorln(v ...interface{}) {
	defaultLogger.output(ErrorLevel, 0, fmt.Sprintln(v...))
}

func Errorf(format string, v ...interface{}) {
	defaultLogger.output(ErrorLevel, 0, fmt.Sprintf(format, v...))
}

func Fatal(v ...interface{}) {
	defaultLogger.output(FatalLevel, 0, fmt.Sprint(v...))
	defaultLogger.Close()
	os.Exit(1)
}

func FatalDepth(depth int, v ...interface{}) {
	defaultLogger.output(FatalLevel, depth, fmt.Sprint(v...))
	defaultLogger.Close()
	os.Exit(1)
}

func Fatalln(v ...interface{}) {
	defaultLogger.output(FatalLevel, 0, fmt.Sprintln(v...))
	defaultLogger.Close()
	os.Exit(1)
}

func Fatalf(format string, v ...interface{}) {
	defaultLogger.output(FatalLevel, 0, fmt.Sprintf(format, v...))
	defaultLogger.Close()
	os.Exit(1)
}

func (l *Logger) output(s Level, depth int, txt string) {
	logLock.Lock()
	var switchFrameLvl = 3
	switch {
	case s == FatalLevel && l.outputLevel >= FatalLevel:
		l.inner.Output(switchFrameLvl+depth, "FATAL: "+txt)
	case s == ErrorLevel && l.outputLevel >= ErrorLevel:
		l.inner.Output(switchFrameLvl+depth, "ERROR: "+txt)
	case s == WarnLevel && l.outputLevel >= WarnLevel:
		l.inner.Output(switchFrameLvl+depth, "WARN: "+txt)
	case s == InfoLevel && l.outputLevel >= InfoLevel:
		l.inner.Output(switchFrameLvl+depth, "INFO: "+txt)
	case s == DebugLevel && l.outputLevel >= DebugLevel:
		l.inner.Output(switchFrameLvl+depth, "DEBUG: "+txt)
	}
	logLock.Unlock()
}
