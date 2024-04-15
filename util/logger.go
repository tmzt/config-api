package util

import (
	"fmt"
	"log"
	"runtime"
	"strings"

	restful "github.com/emicklei/go-restful/v3"
)

type LoggerInterface interface {
	Println(v ...interface{})
	Printf(format string, v ...interface{})

	Info(v ...interface{})
	Infof(format string, v ...interface{})

	Error(v ...interface{})
	Errorf(format string, v ...interface{})

	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
}

type SetRequestLogger interface {
	LoggerInterface
	SetRequest(request *restful.Request)
}

type Logger struct {
	logger  *log.Logger
	prefix  string
	request *restful.Request
}

func NewLogger(prefix string, extraFlags int) *Logger {
	// flags := log.LstdFlags | log.Lshortfile | extraFlags
	flags := log.LstdFlags | extraFlags
	logger := log.New(log.Writer(), prefix+": ", flags)

	return &Logger{
		logger: logger,
		prefix: prefix,
	}
}

func (l *Logger) SetRequest(request *restful.Request) {
	l.request = request
}

func (l *Logger) buildPrefix() string {
	// Two frames up to get the caller of the caller
	_, filename, line, _ := runtime.Caller(2)

	// prefix := l.prefix + " " + filename + ":" + string(line)

	sb := strings.Builder{}

	if l.request != nil {
		if amzRequestId := l.request.Request.Header.Get("X-Amzn-Trace-Id"); amzRequestId != "" {
			// return amzRequestId + " " + l.prefix
			// prefix = amzRequestId + " " + prefix
			sb.WriteString(amzRequestId + " ")
		}
	}

	sb.WriteString(l.prefix)
	sb.WriteString(" ")

	sb.WriteString(fmt.Sprintf("%s:%d ", filename, line))

	return sb.String()
}

func (l *Logger) Println(v ...interface{}) {
	prefix := l.buildPrefix()
	out := prefix + " " + fmt.Sprint(v...)
	l.logger.Println(out)
}

func (l *Logger) Printf(format string, v ...interface{}) {
	prefix := l.buildPrefix()
	out := prefix + " " + fmt.Sprintf(format, v...)
	// l.logger.Printf(prefix+format, v...)
	l.logger.Print(out)
}

func (l *Logger) Info(v ...interface{}) {
	prefix := l.buildPrefix()
	out := prefix + " " + fmt.Sprintln(v...)
	l.logger.Print(out)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	prefix := l.buildPrefix()
	out := prefix + " " + fmt.Sprintf(format, v...)
	l.logger.Print(out)
}

func (l *Logger) Error(v ...interface{}) {
	prefix := l.buildPrefix()
	out := prefix + " " + fmt.Sprintln(v...)
	l.logger.Print(out)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	prefix := l.buildPrefix()
	out := prefix + " " + fmt.Sprintf(format, v...)
	l.logger.Print(out)
}

func (l *Logger) Fatal(v ...interface{}) {
	prefix := l.buildPrefix()
	out := prefix + " " + fmt.Sprintln(v...)
	l.logger.Fatal(out)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	prefix := l.buildPrefix()
	out := prefix + " " + fmt.Sprintf(format, v...)
	l.logger.Fatal(out)
}
