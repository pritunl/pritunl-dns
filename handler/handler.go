package handler

import (
	"github.com/miekg/dns"
	"github.com/pritunl/pritunl-dns/question"
	"github.com/pritunl/pritunl-dns/resolver"
	"net"
	"time"
)

type Handler struct {
	reslvr *resolver.Resolver
}

func (h *Handler) handle(proto string, w dns.ResponseWriter, r *dns.Msg) {
	ques := question.NewQuestion(r.Question[0])

	if ques.IsIpQuery {
		// TODO
		if ques.NameTrim == "test.pritunl.com" {
			msg := &dns.Msg{}
			msg.SetReply(r)

			switch ques.Qclass {
			case dns.TypeA:
				header := dns.RR_Header{
					Name:   ques.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    5,
				}
				record := &dns.A{
					Hdr: header,
					A:   net.ParseIP("10.0.0.10"),
				}
				msg.Answer = append(msg.Answer, record)
			case dns.TypeAAAA:
				header := dns.RR_Header{
					Name:   ques.Name,
					Rrtype: dns.TypeAAAA,
					Class:  dns.ClassINET,
					Ttl:    5,
				}
				record := &dns.AAAA{
					Hdr:  header,
					AAAA: net.ParseIP("10.0.0.10"),
				}
				msg.Answer = append(msg.Answer, record)
			}

			w.WriteMsg(msg)
			return
		}

		if ques.NameTrim == "error" {
			dns.HandleFailed(w, r)
			return
		}
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
		},
	}

	return
}
