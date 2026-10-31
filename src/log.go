package opennox

import (
	"bufio"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/opennox/libs/env"
	"github.com/opennox/libs/log"
)

var (
	logFile    io.Closer
	logBuf     bufio.Writer
	logFileHnd = log.NewTextHandler(nil)
)

func init() {
	logFileHnd.SetLevel(slog.LevelDebug)
	log.AddHandler(logFileHnd)
}

func closeLog() {
	setLogFile(nil)
}

func setLogFile(w io.WriteCloser) {
	if f := logFile; f != nil {
		defer func() {
			logBuf.Flush()
			f.Close()
		}()
	}
	if w == nil {
		logFile = nil
		logFileHnd.SetWriter(nil)
		return
	}
	logFile = w
	logBuf.Reset(w)
	logFileHnd.SetWriter(&logBuf)
}

func defaultLogDir() string {
	logDir := filepath.Dir(os.Args[0])
	if sdir := env.AppUserDir(); sdir != "" {
		logDir = sdir
	}
	return filepath.Join(logDir, "logs")
}

func writeLogsToDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	name := "opennox.log"
	if isDedicatedServer {
		name = "opennox-server.log"
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	setLogFile(f)
	// the game runs without a console, so write crash messages to a file, keeping the earlier crashes
	if cf, err := os.OpenFile(filepath.Join(dir, "crash.txt"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		if err := debug.SetCrashOutput(cf, debug.CrashOptions{}); err != nil {
			slog.Error("cannot set crash output", "err", err)
		}
		cf.Close() // SetCrashOutput keeps its own copy of the file
	}
	return nil
}
