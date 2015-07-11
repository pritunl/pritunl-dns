package resolver

import (
	"github.com/dropbox/godropbox/errors"
	"github.com/miekg/dns"
	"github.com/pritunl/pritunl-dns/utils"
	"time"
)

type Resolver struct {
	Timeout  time.Duration
	Interval time.Duration
	Servers  []string
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
