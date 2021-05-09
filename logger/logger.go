package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

func Init() {
	logrus.SetFormatter(&formatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)
}
