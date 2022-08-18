package logging

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"
)

type Logging struct {
	logger   *log.Logger
	template *template.Template
}

type LogFormat struct {
	RemoteAddr     string
	TimeLocal      string
	RequestMethod  string
	ServerProtocol string
	Status         int
	BodyBytesSent  int
	HttpReferer    string
	HttpUserAgent  string
}

type LogConfig struct {
	Output   string
	Format   string
	FilePath string
}

const defaultLogFormat = `{{.RemoteAddr}} [{{.TimeLocal}}] "{{.RequestMethod}} {{.ServerProtocol}}" {{.Status}} {{.BodyBytesSent}} "{{.HttpReferer}}" "{{.HttpUserAgent}}"`

func New(logconfig *LogConfig) (*Logging, error) {
	var err error
	logging := &Logging{}

	if logging.logger, err = buildLogger(logconfig); err != nil {
		return nil, err
	}

	if logging.template, err = buildLogFormatTemplate(logconfig); err != nil {
		return nil, err
	}

	return logging, nil
}

func (l *Logging) Write(lf LogFormat) error {
	if l == nil || l.logger == nil {
		return nil
	}

	wr := new(bytes.Buffer)
	if err := l.template.Execute(wr, lf); err != nil {
		return err
	}

	l.logger.Println(wr.String())
	return nil
}

func (l *Logging) WriteHTTPLog(w http.ResponseWriter, r *http.Request, status int, contentLength int) error {
	t := time.Now()
	lf := LogFormat{RemoteAddr: r.RemoteAddr, TimeLocal: t.Format("02/Jan/2006:15:04:05 -0700"), RequestMethod: r.Method, ServerProtocol: r.Proto, Status: status, BodyBytesSent: contentLength, HttpReferer: r.Referer(), HttpUserAgent: r.UserAgent()}
	return l.Write(lf)
}

func buildLogger(logconfig *LogConfig) (*log.Logger, error) {
	switch logconfig.Output {
	case "stdout":
		return log.New(os.Stdout, "", 0), nil
	case "":
		return log.New(os.Stdout, "", 0), nil
	case "stderr":
		return log.New(os.Stderr, "", 0), nil
	case "discard":
		return nil, nil
	case "file":
		f, err := os.OpenFile(logconfig.FilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			return nil, err
		}
		return log.New(f, "", 0), nil
	default:
		return nil, fmt.Errorf("log format is invalid value: %s", logconfig.Output)
	}
}

func buildLogFormatTemplate(logconfig *LogConfig) (*template.Template, error) {
	format := logconfig.Format
	if len(format) == 0 {
		format = defaultLogFormat
	}

	return template.New("logformat").Parse(format)
}
