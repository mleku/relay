package ratel

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"

	"github.com/fatih/color"

	"relay.mleku.dev/log"
	"relay.mleku.dev/lol"
)

// NewLogger creates a new badger logger.
func NewLogger(logLevel int32, label string) (l *logger) {
	log.T.Ln("getting logger for", label)
	l = &logger{Label: color.New(color.Bold).Sprint(label)}
	l.Log, _, _ = lol.New(os.Stderr, 4)
	l.Level.Store(logLevel)
	return
}

type logger struct {
	Level atomic.Int32
	Label string
	*lol.Log
}

// SetLogLevel atomically adjusts the log level to the given log level code.
func (l *logger) SetLogLevel(level int) {
	l.Level.Store(int32(level))
}

// Errorf is a log printer for this level of message.
func (l *logger) Errorf(s string, i ...interface{}) {
	if l.Level.Load() >= lol.Error {
		s = l.Label + ": " + s
		txt := fmt.Sprintf(s, i...)
		l.Log.E.Ln(strings.TrimSpace(txt))
	}
}

// Warningf is a log printer for this level of message.
func (l *logger) Warningf(s string, i ...interface{}) {
	if l.Level.Load() >= lol.Warn {
		s = l.Label + ": " + s
		txt := fmt.Sprintf(s, i...)
		l.Log.W.F(strings.TrimSpace(txt))
	}
}

// Infof is a log printer for this level of message.
func (l *logger) Infof(s string, i ...interface{}) {
	if l.Level.Load() >= lol.Info {
		s = l.Label + ": " + s
		txt := fmt.Sprintf(s, i...)
		l.Log.I.F(strings.TrimSpace(txt))
	}
}

// Debugf is a log printer for this level of message.
func (l *logger) Debugf(s string, i ...interface{}) {
	if l.Level.Load() >= lol.Trace {
		s = l.Label + ": " + s
		txt := fmt.Sprintf(s, i...)
		l.Log.D.F(strings.TrimSpace(txt))
	}
}
