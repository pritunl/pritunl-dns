package server

import (
	"fmt"
	"github.com/dropbox/godropbox/errors"
	"github.com/miekg/dns"
	"github.com/pritunl/pritunl-dns/handler"
	"time"
)

type Server struct {
	Host     string
	Port     int
	Timeout  time.Duration
	Interval time.Duration
}

func (s *Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

func (s *Server) Run() (err error) {
	hndlr := handler.NewHandler(s.Timeout, s.Interval)

	tcpHandler := dns.NewServeMux()
	tcpHandler.HandleFunc(".", hndlr.HandleTcp)
	tcpServer := &dns.Server{
		Addr:         s.Addr(),
		Net:          "tcp",
		Handler:      tcpHandler,
		ReadTimeout:  s.Timeout,
		WriteTimeout: s.Timeout,
	}
	go func() {
		err := tcpServer.ListenAndServe()
		if err != nil {
			panic(err)
		}

		err = &ServerError{
			errors.Wrap(err, "server: Unexpected TCP server exit"),
		}
		panic(err)
	}()

	udpHandler := dns.NewServeMux()
	udpHandler.HandleFunc(".", hndlr.HandleUdp)
	udpServer := &dns.Server{
		Addr:         s.Addr(),
		Net:          "udp",
		Handler:      udpHandler,
		ReadTimeout:  s.Timeout,
		WriteTimeout: s.Timeout,
	}
	go func() {
		err := udpServer.ListenAndServe()
		if err != nil {
			panic(err)
		}

		err = &ServerError{
			errors.Wrap(err, "server: Unexpected UDP server exit"),
		}
		panic(err)
	}()

	return
}
