package logging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"text/template"
	"time"
)

type Logging struct {
	logger   *log.Logger
	template *template.Template
	escape   string
	mu       sync.Mutex
	file     *os.File
	filePath string
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
	Escape   string
}

type LogEscape int

const defaultLogFormat = `{{.RemoteAddr}} [{{.TimeLocal}}] "{{.RequestMethod}} {{.ServerProtocol}}" {{.Status}} {{.BodyBytesSent}} "{{.HttpReferer}}" "{{.HttpUserAgent}}"`

func New(logconfig *LogConfig) (*Logging, error) {
	var err error
	logging := &Logging{filePath: logconfig.FilePath}

	if logging.logger, logging.file, err = buildLogger(logconfig); err != nil {
		return nil, err
	}

	if logging.template, err = buildLogFormatTemplate(logconfig); err != nil {
		return nil, err
	}

	if logging.escape, err = buildLogEscape(logconfig); err != nil {
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

	msg := wr.String()
	if l.escape == "json" {
		b, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		msg = string(b)
	}

	l.mu.Lock()
	l.logger.Println(msg)
	l.mu.Unlock()
	return nil
}

func (l *Logging) WriteHTTPLog(w http.ResponseWriter, r *http.Request, status int, contentLength int) error {
	t := time.Now()
	lf := LogFormat{RemoteAddr: r.RemoteAddr, TimeLocal: t.Format("02/Jan/2006:15:04:05 -0700"), RequestMethod: r.Method, ServerProtocol: r.Proto, Status: status, BodyBytesSent: contentLength, HttpReferer: r.Referer(), HttpUserAgent: r.UserAgent()}
	return l.Write(lf)
}

func (l *Logging) Reopen() error {
	if l.file == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.file.Sync(); err != nil {
		return err
	}
	if err := l.file.Close(); err != nil {
		return err
	}
	f, err := buildLogFile(l.filePath)
	if err != nil {
		return err
	}
	l.logger = log.New(f, "", 0)
	l.file = f
	return nil
}

func buildLogger(logconfig *LogConfig) (*log.Logger, *os.File, error) {
	switch logconfig.Output {
	case "stdout":
		return log.New(os.Stdout, "", 0), nil, nil
	case "":
		return log.New(os.Stdout, "", 0), nil, nil
	case "stderr":
		return log.New(os.Stderr, "", 0), nil, nil
	case "discard":
		return nil, nil, nil
	case "file":
		f, err := buildLogFile(logconfig.FilePath)
		if err != nil {
			return nil, nil, err
		}
		return log.New(f, "", 0), f, nil
	default:
		return nil, nil, fmt.Errorf("log format is invalid value: %s", logconfig.Output)
	}
}

func buildLogFormatTemplate(logconfig *LogConfig) (*template.Template, error) {
	format := logconfig.Format
	if len(format) == 0 {
		format = defaultLogFormat
	}

	return template.New("logformat").Parse(format)
}

func buildLogEscape(logconfig *LogConfig) (string, error) {
	if len(logconfig.Escape) == 0 {
		return "", nil
	}

	if logconfig.Escape != "json" {
		return "", fmt.Errorf("log escape is invalid value: %s", logconfig.Escape)
	}

	return logconfig.Escape, nil
}

func buildLogFile(path string) (*os.File, error) {
	f, err := os.OpenFile(filepath.Clean(path), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}

	return f, nil
}
