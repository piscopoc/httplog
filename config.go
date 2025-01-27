package httplog

import (
	"os"
	"strings"
	"time"

	"golang.org/x/exp/slog"
)

var DefaultOptions = Options{
	LogLevel:        "info",
	LevelFieldName:  "level",
	JSON:            false,
	Concise:         false,
	Tags:            nil,
	SkipHeaders:     nil,
	QuietDownRoutes: nil,
	QuietDownPeriod: 0,
	TimeFieldFormat: time.RFC3339Nano,
	TimeFieldName:   "timestamp",
}

type Options struct {
	// LogLevel defines the minimum level of severity that app should log.
	// Must be one of:
	// "debug", "info", "warn", "error"
	LogLevel string

	// LevelFieldName sets the field name for the log level or severity.
	// Some providers parse and search for different field names.
	LevelFieldName string

	// JSON enables structured logging output in json. Make sure to enable this
	// in production mode so log aggregators can receive data in parsable format.
	//
	// In local development mode, its appropriate to set this value to false to
	// receive pretty output and stacktraces to stdout.
	JSON bool

	// Concise mode includes fewer log details during the request flow. For example
	// excluding details like request content length, user-agent and other details.
	// This is useful if during development your console is too noisy.
	Concise bool

	// Tags are additional fields included at the root level of all logs.
	// These can be useful for example the commit hash of a build, or an environment
	// name like prod/stg/dev
	Tags map[string]string

	// SkipHeaders are additional headers which are redacted from the logs
	SkipHeaders []string

	// QuietDownRoutes are routes which are temporarily excluded from logging for a QuietDownPeriod after it occurs
	// for the first time
	// to cancel noise from logging for routes that are known to be noisy.
	QuietDownRoutes []string

	// QuietDownPeriod is the duration for which a route is excluded from logging after it occurs for the first time
	// if the route is in QuietDownRoutes
	QuietDownPeriod time.Duration

	// TimeFieldFormat defines the time format of the Time field, defaulting to "time.RFC3339Nano" see options at:
	// https://pkg.go.dev/time#pkg-constants
	TimeFieldFormat string

	// TimeFieldName sets the field name for the time field.
	// Some providers parse and search for different field names.
	TimeFieldName string

	// SourceFieldName sets the field name for the source field which logs
	// the location where the logger was called
	// its "" if not enabled
	SourceFieldName string
}

// Take the string representation of the log level and turn that into a compatible slog.Level
// of underlying zerolog pkg and its global logger.
func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Configure will set new global/default options for the httplog and behaviour
// of underlying zerolog pkg and its global logger.
func Configure(opts Options) {
	// if opts.LogLevel is not set
	// it would be 0 which is LevelInfo

	if opts.LevelFieldName == "" {
		opts.LevelFieldName = "level"
	}

	if opts.TimeFieldFormat == "" {
		opts.TimeFieldFormat = time.RFC3339Nano
	}

	if opts.TimeFieldName == "" {
		opts.TimeFieldName = "timestamp"
	}

	if len(opts.QuietDownRoutes) > 0 {
		if opts.QuietDownPeriod == 0 {
			opts.QuietDownPeriod = 5 * time.Minute
		}
	}

	// Pre-downcase all SkipHeaders
	for i, header := range opts.SkipHeaders {
		opts.SkipHeaders[i] = strings.ToLower(header)
	}

	DefaultOptions = opts

	var addSource bool
	if opts.SourceFieldName != "" {
		addSource = true
	}

	replaceAttrs := func(_ []string, a slog.Attr) slog.Attr {
		switch a.Key {
		case slog.LevelKey:
			a.Key = opts.LevelFieldName
		case slog.TimeKey:
			a.Key = opts.TimeFieldName
			a.Value = slog.StringValue(a.Value.Time().Format(opts.TimeFieldFormat))
		case slog.SourceKey:
			if opts.SourceFieldName != "" {
				a.Key = opts.SourceFieldName
			}
		}
		return a
	}

	handlerOpts := &slog.HandlerOptions{
		Level:       parseLogLevel(opts.LogLevel),
		ReplaceAttr: replaceAttrs,
		AddSource:   addSource,
	}

	if !opts.JSON {
		slog.SetDefault(slog.New(NewPrettyHandler(os.Stdout, handlerOpts)))
	} else {
		slog.SetDefault(slog.New(handlerOpts.NewJSONHandler(os.Stderr)))
	}
}
