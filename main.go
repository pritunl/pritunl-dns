package main

import (
	"github.com/pritunl/pritunl-dns/server"
	"os"
	"os/signal"
	"time"
)

func main() {
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
