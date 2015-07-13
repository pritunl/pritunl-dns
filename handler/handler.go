package handler

import (
	"github.com/miekg/dns"
	"github.com/pritunl/pritunl-dns/question"
	"github.com/pritunl/pritunl-dns/resolver"
	"net"
	"time"
	"strings"
)

type Handler struct {
	reslvr *resolver.Resolver
}

func (h *Handler) handle(proto string, w dns.ResponseWriter, r *dns.Msg) {
	ques := question.NewQuestion(r.Question[0])

	if ques.IsIpQuery && strings.HasSuffix(ques.NameTrim, ".vpn") {
		msg, err := h.reslvr.LookupUser(ques, r)
		if err != nil {
			dns.HandleFailed(w, r)
			return
		}
		w.WriteMsg(msg)
	}

	res, err := h.reslvr.Lookup(proto, r)
	if err != nil {
		dns.HandleFailed(w, r)
		return
	}

	w.WriteMsg(res)
}

func (h *Handler) HandleTcp(w dns.ResponseWriter, r *dns.Msg) {
	h.handle("tcp", w, r)
}

func (h *Handler) HandleUdp(w dns.ResponseWriter, r *dns.Msg) {
	h.handle("udp", w, r)
}

func NewHandler(timeout, interval time.Duration) (h *Handler) {
	h = &Handler{
		reslvr: &resolver.Resolver{
			Timeout:  timeout,
			Interval: interval,
			Servers: []string{"8.8.8.8:53", "8.8.4.4:53"},
		},
	}

	return
}
