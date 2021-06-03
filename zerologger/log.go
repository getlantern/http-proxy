package zerologger

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/getlantern/ops"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

type LoggerWrapper struct {
	zerolog.Logger
	printStack bool
}

var logger zerolog.Logger

func init() {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	logger = zerolog.New(os.Stdout).With().Timestamp().CallerWithSkipFrameCount(3).Logger()
}

func Named(name string) LoggerWrapper {
	printStack, _ := strconv.ParseBool(os.Getenv("PRINT_STACK"))
	printJSON, _ := strconv.ParseBool(os.Getenv("PRINT_JSON"))
	namedLogger := logger.With().Str("component", name).Logger()
	if !printJSON {
		namedLogger = namedLogger.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}
	return LoggerWrapper{namedLogger, printStack}
}

func (l LoggerWrapper) Debug(arg interface{}) {
	l.Logger.Debug().Msg(fmt.Sprintf("%v", arg))

}

func (l LoggerWrapper) Debugf(message string, args ...interface{}) {
	l.Logger.Debug().Msg(fmt.Sprintf(message, args...))
}

func (l LoggerWrapper) logError(message string, arg interface{}) error {
	var err error
	switch e := arg.(type) {
	case error:
		err = e
	default:
		err = fmt.Errorf("%v", e)
	}
	log := l.Logger.Error().Interface("ops", ops.AsMap(err, false))
	if l.printStack {
		log = log.Stack().Err(errors.WithStack(err))
	} else {
		log = log.Err(err)
	}
	log.Msg(message)
	return err
}
func (l LoggerWrapper) Error(arg interface{}) error {
	return l.logError("", arg)
}

func (l LoggerWrapper) Errorf(message string, args ...interface{}) error {
	if err, ok := args[len(args)-1].(error); ok {
		// small hack to clean Errorf calls of the formatting suffix
		if strings.HasSuffix(message, ": %v") {
			message = message[0 : len(message)-4]
		}
		return l.logError(message, err)
	} else {
		l.Logger.Error().Msg(fmt.Sprintf(message, args...))
	}
	return nil
}

func (l LoggerWrapper) Fatal(arg interface{}) {
	l.Logger.Fatal().Msg(fmt.Sprintf("%v", arg))
}

func (l LoggerWrapper) Fatalf(message string, args ...interface{}) {
	l.Logger.Fatal().Msg(fmt.Sprintf(message, args...))
}

func (l LoggerWrapper) Trace(arg interface{}) {
	l.Logger.Trace().Msg(fmt.Sprintf("%v", arg))
}

func (l LoggerWrapper) Tracef(message string, args ...interface{}) {
	l.Logger.Trace().Msg(fmt.Sprintf(message, args...))
}

func (l LoggerWrapper) IsTraceEnabled() bool {
	return true
}
