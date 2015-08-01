package resolver

import (
	"crypto/md5"
	"github.com/dropbox/godropbox/errors"
	"github.com/miekg/dns"
	"github.com/pritunl/pritunl-dns/database"
	"github.com/pritunl/pritunl-dns/question"
	"github.com/pritunl/pritunl-dns/utils"
	"labix.org/v2/mgo/bson"
	"net"
	"strings"
	"time"
)

type Client struct {
	Id          bson.ObjectId `bson:"_id"`
	Domain      bson.Binary   `bson:"domain"`
	Network     string        `bson:"network"`
	VirtAddress string        `bson:"virt_address"`
}

type Resolver struct {
	Timeout  time.Duration
	Interval time.Duration
	Servers  []string
}

func (r *Resolver) LookupUser(ques *question.Question, subnet string,
	req *dns.Msg) (msg *dns.Msg, err error) {

	if ques.Qclass != dns.TypeA {
		err = &NotFoundError{
			errors.New("resolver: Invalid dns type"),
		}
		return
	}

	db := database.GetDatabase()
	coll := db.Clients()

	key := md5.Sum([]byte(ques.Domain))
	cursor := coll.Find(bson.M{
		"domain": bson.Binary{
			Kind: 0x05,
			Data: key[:],
		},
	}).Iter()

	ipStr := ""
	clnt := Client{}
	for cursor.Next(&clnt) {
		ipStr = clnt.VirtAddress
		if clnt.Network == subnet {
			break
		}
	}
	ipStr = strings.Split(ipStr, "/")[0]

	if ipStr == "" {
		err = &UnknownError{
			errors.New("resolver: Failed to find ip"),
		}
		return
	}

	ip := net.ParseIP(ipStr)
	if err != nil {
		err = &UnknownError{
			errors.Wrap(err, "resolver: Unknown parse error"),
		}
	}

	msg = &dns.Msg{}
	msg.SetReply(req)

	header := dns.RR_Header{
		Name:   ques.Name,
		Rrtype: dns.TypeA,
		Class:  dns.ClassINET,
		Ttl:    5,
	}
	record := &dns.A{
		Hdr: header,
		A:   ip,
	}
	msg.Answer = append(msg.Answer, record)

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
