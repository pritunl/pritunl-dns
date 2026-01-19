package server

import (
	"fmt"
	"time"

	"github.com/dropbox/godropbox/errors"
	"github.com/miekg/dns"
	"github.com/pritunl/pritunl-dns/handler"
)

type Server struct {
	Port    int
	Timeout time.Duration
}

func (s *Server) Addr() string {
	return fmt.Sprintf(":%d", s.Port)
}

func (s *Server) Run() (err error) {
	hndlr := handler.NewHandler(s.Timeout)
	dns.HandleFunc(".", hndlr.Handle)

	tcpServer := &dns.Server{
		Addr:         s.Addr(),
		Net:          "tcp",
		ReadTimeout:  s.Timeout,
		WriteTimeout: s.Timeout,
	}
	go func() {
		e := tcpServer.ListenAndServe()
		if e != nil {
			panic(e)
		}

		e = &ServerError{
			errors.Wrap(e, "server: Unexpected TCP server exit"),
		}
		panic(e)
	}()

	udpServer := &dns.Server{
		Addr:         s.Addr(),
		Net:          "udp",
		ReadTimeout:  s.Timeout,
		WriteTimeout: s.Timeout,
	}
	go func() {
		e := udpServer.ListenAndServe()
		if e != nil {
			panic(e)
		}

		e = &ServerError{
			errors.Wrap(e, "server: Unexpected UDP server exit"),
		}
		panic(e)
	}()

	return
}
