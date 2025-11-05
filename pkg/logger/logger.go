package logger

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	log *logrus.Logger
	// L is an alias for the standard logger.
	L = logrus.NewEntry(NewLogger())
)

type (
	loggerKey struct{}

	// Fields type to pass to `WithFields`, alias from `logrus`.
	Fields = logrus.Fields
)

func Field(fields map[string]interface{}) Fields {
	return Fields(fields)
}

// FromContext gets logger from context.
// In case of no logger in context, the default logger is returned.
func FromContext(ctx context.Context) *logrus.Entry {
	log := ctx.Value(loggerKey{})
	if log == nil {
		return L.WithContext(ctx)
	}
	return log.(*logrus.Entry)
}

// WithLogger returns a new context with the provided logger. Use in
// combination with logger.WithField(s) for great effect.
func WithLogger(ctx context.Context, logger *logrus.Entry) context.Context {
	e := logger.WithContext(ctx)
	return context.WithValue(ctx, loggerKey{}, e)
}

// NewLogger returns a new logger.
func NewLogger() *logrus.Logger {
	if log == nil {
		log = &logrus.Logger{
			Out:          os.Stderr,
			Formatter:    &logrus.JSONFormatter{},
			Hooks:        make(logrus.LevelHooks),
			ReportCaller: true,
			Level:        logrus.DebugLevel,
		}
	}
	return log
}

type EasyFormatter struct{}

// The Formatter interface is used to implement a custom Formatter. It takes an
// `Entry`. It exposes all the fields, including the default ones:
//
// * `entry.Data["msg"]`. The message passed from Info, Warn, Error ..
// * `entry.Data["time"]`. The timestamp.
// * `entry.Data["level"]. The level the entry was logged at.
//
// Any additional fields added with `WithField` or `WithFields` are also in
// `entry.Data`. Format is expected to return an array of bytes which are then
// logged to `logger.Out`.
func (f *EasyFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var msg bytes.Buffer

	msg.WriteString(entry.Time.Format("2006-01-02T15:04:05Z"))
	msg.WriteRune('\t')
	msg.WriteString(strings.ToUpper(entry.Level.String()))
	msg.WriteRune('\t')

	// filename with pkg name stripped
	fileName := entry.Caller.File[strings.LastIndex(entry.Caller.File, "/")+1:]
	msg.WriteString(fileName)
	msg.WriteRune(':')
	msg.WriteString(strconv.Itoa(entry.Caller.Line))
	msg.WriteRune('\t')
	msg.WriteString(entry.Message)
	msg.WriteRune('\t')
	if len(entry.Data) > 0 {
		msg.WriteString(fmt.Sprintf("%v", entry.Data))
	}
	msg.WriteRune('\n')
	return msg.Bytes(), nil
}
