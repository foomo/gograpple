package log

import (
	"io"

	"github.com/sirupsen/logrus"
)

func Writer(src string) io.Writer {
	return logrus.New().WithField("src", src).Writer()
}
