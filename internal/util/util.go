package util

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

type Logger struct {
	nc       *nats.Conn
	subject  string
	nodeName string
}

func NewLogger(nodeName string, deviceCategory string, nc *nats.Conn) *Logger {
	subject := fmt.Sprintf("smartie.%s.%s.logs", deviceCategory, nodeName)
	return &Logger{
		nc:       nc,
		subject:  subject,
		nodeName: nodeName,
	}
}

func (l *Logger) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (l *Logger) Fire(entry *logrus.Entry) error {
	entry.Data["node"] = l.nodeName

	msg, err := entry.Bytes()

	if err == nil {
		l.nc.Publish(l.subject, msg)

	} else {
		l.nc.Publish(l.subject, []byte(err.Error()))
		return err
	}
	return nil
}

func Fatal(err error) {
	if err != nil {
		logrus.Fatal(err)
	}
}
