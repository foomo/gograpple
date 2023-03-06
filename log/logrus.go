package log

import (
	"io"

	"github.com/sirupsen/logrus"
)

func Writer(src string) io.Writer {
	return Entry(src).Writer()
}

func Entry(component string) *logrus.Entry {
	if component != "" {
		return logrus.StandardLogger().WithField("component", component)
	}
	return logrus.NewEntry(logrus.StandardLogger())
}

func Logger() *logrus.Logger {
	return logrus.StandardLogger()
}
