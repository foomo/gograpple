package log

import (
	"io"

	"github.com/sirupsen/logrus"
)

func Writer(src string) io.Writer {
	return Entry(src).Writer()
}

func Entry(src string) *logrus.Entry {
	return logrus.StandardLogger().WithField("src", src)
}
