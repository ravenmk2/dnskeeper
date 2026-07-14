package log

import (
	"io"

	"github.com/sirupsen/logrus"
)

func Setup(level string, out io.Writer) *logrus.Logger {
	l := logrus.New()
	l.SetOutput(out)
	if level != "" {
		if lvl, err := logrus.ParseLevel(level); err == nil {
			l.SetLevel(lvl)
		}
	}
	l.SetFormatter(&logrus.JSONFormatter{
		DisableHTMLEscape: true,
	})
	return l
}
