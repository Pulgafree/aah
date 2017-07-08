// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package log implements a simple, flexible, non-blocking logger.
// It supports `console`, `file` (rotation by daily, size, lines).
// It also has a predefined 'standard' Logger accessible through helper
// functions `Error{f}`, `Warn{f}`, `Info{f}`, `Debug{f}`, `Trace{f}`,
// `Print{f,ln}`, `Fatal{f,ln}`, `Panic{f,ln}` which are easier to use than creating
// a Logger manually. Default logger writes to standard error and prints log
// `Entry` details as per `DefaultPattern`.
//
// aah log package can be used as drop-in replacement for standard go logger
// with features.
//
// 	log.Info("Welcome ", "to ", "aah ", "logger")
// 	log.Infof("%v, %v, %v", "simple", "flexible", "non-blocking logger")
//
// 	// Output:
// 	2016-07-03 19:22:11.504 INFO  Welcome to aah logger
// 	2016-07-03 19:22:11.504 INFO  simple, flexible, non-blocking logger
package log

import (
	"errors"
	"fmt"
	"io"
	slog "log"
	"os"
	"strings"
	"sync"
	"time"

	"aahframework.org/config.v0"
)

// Level type definition
type level uint8

// Log Level definition
const (
	levelFatal level = iota
	levelPanic
	LevelError
	LevelWarn
	LevelInfo
	LevelDebug
	LevelTrace
	LevelUnknown
)

var (
	// Version no. of aahframework.org/log library
	Version = "0.5"

	// ErrLogReceiverIsNil returned when suppiled receiver is nil.
	ErrLogReceiverIsNil = errors.New("log: receiver is nil")

	filePermission = os.FileMode(0755)

	// abstract it, can be unit tested
	exit = os.Exit
)

type (
	// Receiver is the interface for pluggable log receiver.
	// For e.g: Console, File, HTTP, etc
	Receiver interface {
		Init(cfg *config.Config) error
		SetPattern(pattern string) error
		IsCallerInfo() bool
		Writer() io.Writer
		Log(e *Entry)
	}

	// Logger is the object which logs the given message into recevier as per deifned
	// format flags. Logger can be used simultaneously from multiple goroutines;
	// it guarantees to serialize access to the Receivers.
	Logger struct {
		cfg      *config.Config
		m        *sync.Mutex
		level    level
		receiver Receiver
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// New method creates the aah logger based on supplied `config.Config`.
func New(cfg *config.Config) (*Logger, error) {
	if cfg == nil {
		return nil, errors.New("log: config is nil")
	}

	logger := &Logger{m: &sync.Mutex{}, cfg: cfg}

	// Receiver
	receiverType := strings.ToUpper(cfg.StringDefault("log.receiver", "CONSOLE"))
	if err := logger.SetReceiver(getReceiverByName(receiverType)); err != nil {
		return nil, err
	}

	// Pattern
	if err := logger.SetPattern(cfg.StringDefault("log.pattern", DefaultPattern)); err != nil {
		return nil, err
	}

	// Level
	if err := logger.SetLevel(cfg.StringDefault("log.level", "DEBUG")); err != nil {
		return nil, err
	}

	return logger, nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Logger methods
//___________________________________

// Level method returns currently enabled logging level.
func (l *Logger) Level() string {
	return levelToLevelName[l.level]
}

// SetLevel method sets the given logging level for the logger.
// For e.g.: INFO, WARN, DEBUG, etc. Case-insensitive.
func (l *Logger) SetLevel(level string) error {
	l.m.Lock()
	defer l.m.Unlock()
	levelFlag := levelByName(level)
	if levelFlag == LevelUnknown {
		return fmt.Errorf("log: unknown log level '%s'", level)
	}
	l.level = levelFlag
	return nil
}

// SetPattern methods sets the log format pattern.
func (l *Logger) SetPattern(pattern string) error {
	l.m.Lock()
	defer l.m.Unlock()
	if l.receiver == nil {
		return ErrLogReceiverIsNil
	}
	return l.receiver.SetPattern(pattern)
}

// SetReceiver sets the given receiver into logger instance.
func (l *Logger) SetReceiver(receiver Receiver) error {
	l.m.Lock()
	defer l.m.Unlock()

	if receiver == nil {
		return ErrLogReceiverIsNil
	}

	l.receiver = receiver
	return l.receiver.Init(l.cfg)
}

// ToGoLogger method wraps the current log writer into Go Logger instance.
func (l *Logger) ToGoLogger() *slog.Logger {
	return slog.New(l.receiver.Writer(), "", slog.LstdFlags)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Logger logging methods
//_______________________________________

// Error logs message as `ERROR`. Arguments handled in the mananer of `fmt.Print`.
func (l *Logger) Error(v ...interface{}) {
	l.output(LevelError, 3, nil, v...)
}

// Errorf logs message as `ERROR`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.output(LevelError, 3, &format, v...)
}

// Warn logs message as `WARN`. Arguments handled in the mananer of `fmt.Print`.
func (l *Logger) Warn(v ...interface{}) {
	l.output(LevelWarn, 3, nil, v...)
}

// Warnf logs message as `WARN`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Warnf(format string, v ...interface{}) {
	l.output(LevelWarn, 3, &format, v...)
}

// Info logs message as `INFO`. Arguments handled in the mananer of `fmt.Print`.
func (l *Logger) Info(v ...interface{}) {
	l.output(LevelInfo, 3, nil, v...)
}

// Infof logs message as `INFO`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Infof(format string, v ...interface{}) {
	l.output(LevelInfo, 3, &format, v...)
}

// Debug logs message as `DEBUG`. Arguments handled in the mananer of `fmt.Print`.
func (l *Logger) Debug(v ...interface{}) {
	l.output(LevelDebug, 3, nil, v...)
}

// Debugf logs message as `DEBUG`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Debugf(format string, v ...interface{}) {
	l.output(LevelDebug, 3, &format, v...)
}

// Trace logs message as `TRACE`. Arguments handled in the mananer of `fmt.Print`.
func (l *Logger) Trace(v ...interface{}) {
	l.output(LevelTrace, 3, nil, v...)
}

// Tracef logs message as `TRACE`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Tracef(format string, v ...interface{}) {
	l.output(LevelTrace, 3, &format, v...)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Logger methods - Drop-in replacement
// for Go standard logger
//_______________________________________

// Print logs message as `INFO`. Arguments handled in the mananer of `fmt.Print`.
func (l *Logger) Print(v ...interface{}) {
	l.output(LevelInfo, 3, nil, v...)
}

// Printf logs message as `INFO`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Printf(format string, v ...interface{}) {
	l.output(LevelInfo, 3, &format, v...)
}

// Println logs message as `INFO`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Println(format string, v ...interface{}) {
	l.output(LevelInfo, 3, &format, v...)
}

// Fatal logs message as `FATAL` and call to os.Exit(1).
func (l *Logger) Fatal(v ...interface{}) {
	l.output(levelFatal, 3, nil, v...)
	exit(1)
}

// Fatalf logs message as `FATAL` and call to os.Exit(1).
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.output(levelFatal, 3, &format, v...)
	exit(1)
}

// Fatalln logs message as `FATAL` and call to os.Exit(1).
func (l *Logger) Fatalln(format string, v ...interface{}) {
	l.output(levelFatal, 3, &format, v...)
	exit(1)
}

// Panic logs message as `PANIC` and call to panic().
func (l *Logger) Panic(v ...interface{}) {
	l.output(levelPanic, 3, nil, v...)
	panic("")
}

// Panicf logs message as `PANIC` and call to panic().
func (l *Logger) Panicf(format string, v ...interface{}) {
	l.output(levelPanic, 3, &format, v...)
	panic(fmt.Sprintf(format, v...))
}

// Panicln logs message as `PANIC` and call to panic().
func (l *Logger) Panicln(format string, v ...interface{}) {
	l.output(levelPanic, 3, &format, v...)
	panic(fmt.Sprintf(format, v...))
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Logger level methods
//___________________________________

// IsLevelInfo method returns true if log level is INFO otherwise false.
func (l *Logger) IsLevelInfo() bool {
	return l.level == LevelInfo
}

// IsLevelError method returns true if log level is ERROR otherwise false.
func (l *Logger) IsLevelError() bool {
	return l.level == LevelError
}

// IsLevelWarn method returns true if log level is WARN otherwise false.
func (l *Logger) IsLevelWarn() bool {
	return l.level == LevelWarn
}

// IsLevelDebug method returns true if log level is DEBUG otherwise false.
func (l *Logger) IsLevelDebug() bool {
	return l.level == LevelDebug
}

// IsLevelTrace method returns true if log level is TRACE otherwise false.
func (l *Logger) IsLevelTrace() bool {
	return l.level == LevelTrace
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

// output method checks the level, formats the arguments and call to configured
// Log receivers.
func (l *Logger) output(level level, calldepth int, format *string, v ...interface{}) {
	if level > l.level {
		return
	}

	entry := acquireEntry()
	defer releaseEntry(entry)
	entry.Time = time.Now()
	entry.Level = level
	if format == nil {
		entry.Message = fmt.Sprint(v...)
	} else {
		entry.Message = fmt.Sprintf(*format, v...)
	}

	if l.receiver.IsCallerInfo() {
		entry.File, entry.Line = fetchCallerInfo(calldepth)
	}

	l.receiver.Log(entry)

	// Execute logger hooks
	go executeHooks(*entry)
}
