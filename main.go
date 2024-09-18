package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pritunl/pritunl-dns/database"
	"github.com/pritunl/pritunl-dns/logger"
	"github.com/pritunl/pritunl-dns/server"
	"github.com/sirupsen/logrus"
)

func main() {
	logger.Init()
	database.Init()

	logrus.WithFields(logrus.Fields{
		"port":     53,
		"protocol": "tcp/udp",
	}).Info("main: Starting DNS server")

	serv := &server.Server{
		Port:     53,
		Timeout:  1 * time.Second,
		Interval: 100 * time.Millisecond,
	}
	serv.Run()

	sig := make(chan os.Signal, 2)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
}
