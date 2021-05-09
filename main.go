package main

import (
	"os"
	"os/signal"
	"time"

	"github.com/pritunl/pritunl-dns/database"
	"github.com/pritunl/pritunl-dns/logger"
	"github.com/pritunl/pritunl-dns/server"
)

func main() {
	logger.Init()
	database.Init()

	serv := &server.Server{
		Port:     53,
		Timeout:  1 * time.Second,
		Interval: 100 * time.Millisecond,
	}
	serv.Run()

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

forever:
	for {
		select {
		case <-sig:
			break forever
		}
	}
}
