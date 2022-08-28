package logging_test

import (
	"os"
	"path"
	"testing"

	"github.com/y-yagi/niwa/internal/logging"
)

func TestWrite(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "niwatest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	logfile := path.Join(tempDir, "niwa.log")
	logger, err := logging.New(&logging.LogConfig{Output: "file", FilePath: logfile})
	if err != nil {
		t.Fatal(err)
	}

	lf := logging.LogFormat{RemoteAddr: "192.168.1.1", TimeLocal: "2022/01/01 00:00", RequestMethod: "GET", ServerProtocol: "https", Status: 200, BodyBytesSent: 0, HttpReferer: "refer", HttpUserAgent: "dummy"}
	if err = logger.Write(lf); err != nil {
		t.Fatal(err)
	}

	log, err := os.ReadFile(logfile)
	if err != nil {
		t.Fatal(err)
	}

	wont := `192.168.1.1 [2022/01/01 00:00] "GET https" 200 0 "refer" "dummy"` + "\n"
	if string(log) != wont {
		t.Errorf("got: %s, wont: %s", log, wont)
	}
}

func TestWrite_WithDiscard(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "niwatest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	logfile := path.Join(tempDir, "niwa.log")
	logger, err := logging.New(&logging.LogConfig{Output: "discard", FilePath: logfile})
	if err != nil {
		t.Fatal(err)
	}

	lf := logging.LogFormat{RemoteAddr: "192.168.1.1", TimeLocal: "2022/01/01 00:00", RequestMethod: "GET", ServerProtocol: "https", Status: 200, BodyBytesSent: 0, HttpReferer: "refer", HttpUserAgent: "dummy"}
	if err = logger.Write(lf); err != nil {
		t.Fatal(err)
	}

	if _, err = os.Stat(logfile); err == nil {
		t.Errorf("expected log file doesn't exist, but it exists")
	}
}

func TestWrite_WithFormat(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "niwatest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	logfile := path.Join(tempDir, "niwa.log")
	logger, err := logging.New(&logging.LogConfig{Output: "file", FilePath: logfile, Format: "RemoteAddr: {{.RemoteAddr}}"})
	if err != nil {
		t.Fatal(err)
	}

	lf := logging.LogFormat{RemoteAddr: "192.168.1.1", TimeLocal: "2022/01/01 00:00", RequestMethod: "GET", ServerProtocol: "https", Status: 200, BodyBytesSent: 0, HttpReferer: "refer", HttpUserAgent: "dummy"}
	if err = logger.Write(lf); err != nil {
		t.Fatal(err)
	}

	log, err := os.ReadFile(logfile)
	if err != nil {
		t.Fatal(err)
	}

	wont := "RemoteAddr: 192.168.1.1\n"
	if string(log) != wont {
		t.Errorf("got: %s, wont: %s", log, wont)
	}
}

func TestWrite_WithJSONEscape(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "niwatest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	logfile := path.Join(tempDir, "niwa.log")
	logger, err := logging.New(&logging.LogConfig{Output: "file", FilePath: logfile, Format: "{ RemoteAddr: {{.RemoteAddr}} }", Escape: "json"})
	if err != nil {
		t.Fatal(err)
	}

	lf := logging.LogFormat{RemoteAddr: "192.168.1.1", TimeLocal: "2022/01/01 00:00", RequestMethod: "GET", ServerProtocol: "https", Status: 200, BodyBytesSent: 0, HttpReferer: "refer", HttpUserAgent: "dummy"}
	if err = logger.Write(lf); err != nil {
		t.Fatal(err)
	}

	log, err := os.ReadFile(logfile)
	if err != nil {
		t.Fatal(err)
	}

	wont := `"{ RemoteAddr: 192.168.1.1 }"` + "\n"
	if string(log) != wont {
		t.Errorf("got:\n \n%s\nwont: \n%s", log, wont)
	}
}

func TestReopen(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "niwatest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	logfile := path.Join(tempDir, "niwa.log")
	logger, err := logging.New(&logging.LogConfig{Output: "file", FilePath: logfile})
	if err != nil {
		t.Fatal(err)
	}

	lf := logging.LogFormat{RemoteAddr: "192.168.1.1", TimeLocal: "2022/01/01 00:00", RequestMethod: "GET", ServerProtocol: "https", Status: 200, BodyBytesSent: 0, HttpReferer: "refer", HttpUserAgent: "dummy"}
	if err = logger.Write(lf); err != nil {
		t.Fatal(err)
	}

	if err := os.Rename(logfile, path.Join(tempDir, "niwa_old.log")); err != nil {
		t.Fatal(err)
	}

	if err := logger.Reopen(); err != nil {
		t.Fatal(err)
	}

	if err = logger.Write(lf); err != nil {
		t.Fatal(err)
	}

	log, err := os.ReadFile(logfile)
	if err != nil {
		t.Fatal(err)
	}

	wont := `192.168.1.1 [2022/01/01 00:00] "GET https" 200 0 "refer" "dummy"` + "\n"
	if string(log) != wont {
		t.Errorf("got: %s, wont: %s", log, wont)
	}
}
