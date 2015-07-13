package resolver

import (
	"github.com/dropbox/godropbox/errors"
	"github.com/miekg/dns"
	"github.com/pritunl/pritunl-dns/question"
	"github.com/pritunl/pritunl-dns/utils"
	"net"
	"time"
)

type Resolver struct {
	Timeout  time.Duration
	Interval time.Duration
	Servers  []string
}

func (r *Resolver) LookupUser(ques question.Question, r *dns.Msg) (
	msg *dns.Msg, err error) {

	if ques.Qclass != dns.TypeA {
		err = &NotFoundError{
			errors.New("resolver: User not found"),
		}
		return
	}

	if ques.NameTrim == "user0.org0.vpn" {
		msg := &dns.Msg{}
		msg.SetReply(r)

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

		return
	}

	err = &NotFoundError{
		errors.New("resolver: User not found"),
	}

	return
}

func (r *Resolver) Lookup(proto string, req *dns.Msg) (
	res *dns.Msg, err error) {

	client := &dns.Client{
		Net:          proto,
		ReadTimeout:  r.Timeout,
		WriteTimeout: r.Timeout,
	}

	resChan := make(chan *dns.Msg, 1)
	waiter := utils.WaitCancel{}
	var ticker *time.Ticker
	var resErr error

	if len(r.Servers) > 2 {
		ticker = time.NewTicker(r.Interval)
	}

	for i, nameserver := range r.Servers {
		if ticker != nil {
			if i != 0 && i%2 == 0 {
				select {
				case res = <-resChan:
					return
				case <-ticker.C:
				}
			}
		}

		waiter.Add(1)

		go func(nameserver string) {
			exRes, _, e := client.Exchange(req, nameserver)
			if e != nil {
				resErr = &ResolveError{
					errors.Wrap(e, "resolver: Socket error"),
				}
				waiter.Done()
				return
			}

			select {
			case resChan <- exRes:
			default:
			}

			waiter.Cancel()
		}(nameserver)
	}

	waiter.Wait()
	select {
	case res = <-resChan:
		return
	default:
		err = resErr
		return
	}

	return
}
