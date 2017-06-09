package server

import (
	"fmt"
	"github.com/dropbox/godropbox/errors"
	"github.com/miekg/dns"
	"github.com/pritunl/pritunl-dns/handler"
	"time"
)

type Server struct {
	Port     int
	Timeout  time.Duration
	Interval time.Duration
}

func (s *Server) Addr() string {
	return fmt.Sprintf(":%d", s.Port)
}

func (s *Server) Run() (err error) {
	hndlr := handler.NewHandler(s.Timeout, s.Interval)
	dns.HandleFunc(".", hndlr.Handle)

	tcpServer := &dns.Server{
		Addr:         s.Addr(),
		Net:          "tcp",
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

	udpServer := &dns.Server{
		Addr:         s.Addr(),
		Net:          "udp",
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
