//go:build !windows

package cmd

import (
	logsyslog "log/syslog"

	"github.com/rs/zerolog"
)

func newSyslogLogger() zerolog.LevelWriter {
	// TODO: log to stderr if this fails?
	syslog, _ := logsyslog.New(logsyslog.LOG_AUTH|logsyslog.LOG_WARNING, "infra-ssh")
	if syslog != nil {
		return zerolog.SyslogLevelWriter(syslog)
	}
	return nil
}
