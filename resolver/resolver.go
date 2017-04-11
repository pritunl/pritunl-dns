package resolver

import (
	"crypto/md5"
	"github.com/dropbox/godropbox/errors"
	"github.com/miekg/dns"
	"github.com/pritunl/pritunl-dns/database"
	"github.com/pritunl/pritunl-dns/question"
	"github.com/pritunl/pritunl-dns/utils"
	"gopkg.in/mgo.v2/bson"
	"net"
	"strings"
	"time"
)

type Client struct {
	Network     string   `bson:"network"`
	DomainName  string   `bson:"domain_name"`
	VirtAddress string   `bson:"virt_address"`
	DnsServers  []string `bson:"dns_servers"`
	DnsSuffix   string   `bson:"dns_suffix"`
}

type Resolver struct {
	Timeout        time.Duration
	Interval       time.Duration
	DefaultServers []string
}

func (r *Resolver) LookupUser(proto string, ques *question.Question,
	subnet string, req *dns.Msg) (msg *dns.Msg, err error) {

	if ques.Qclass != dns.TypeA {
		err = &NotFoundError{
			errors.New("resolver: Invalid dns type"),
		}
		return
	}

	n := utils.LastNthIndex(ques.Domain, ".", 2)

	domain := ques.Domain[n+1:]
	subDomain := ""
	if n > 0 {
		subDomain = ques.Domain[:n]
	}

	db := database.GetDatabase()
	defer db.Close()
	coll := db.Clients()

	key := md5.Sum([]byte(domain))
	cursor := coll.Find(bson.M{
		"domain": bson.Binary{
			Kind: 0x05,
			Data: key[:],
		},
	}).Select(bson.M{
		"network":      1,
		"virt_address": 1,
		"dns_servers":  1,
		"dns_suffix":   1,
	}).Iter()

	clnt := Client{}
	for cursor.Next(&clnt) {
		if clnt.Network == subnet || subnet == "" {
			break
		}
	}

	err = cursor.Close()
	if err != nil {
		err = database.ParseError(err)
		return
	}

	clientIpStr := strings.Split(clnt.VirtAddress, "/")[0]
	if clientIpStr == "" {
		err = &UnknownError{
			errors.New("resolver: Failed to find ip"),
		}
		return
	}

	clientIp := net.ParseIP(clientIpStr)
	if err != nil {
		err = &UnknownError{
			errors.Wrap(err, "resolver: Unknown parse error"),
		}
	}

	if subDomain == "" {
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
			A:   clientIp,
		}
		msg.Answer = make([]dns.RR, 1)
		msg.Answer[0] = record
	} else {
		servers := clnt.DnsServers

		origNames := make([]string, len(req.Question))

		for i, ques := range req.Question {
			n := utils.LastNthIndex(ques.Name, ".", 4)
			if n == -1 {
				err = &ResolveError{
					errors.New("resolver: Failed to parse question name"),
				}
				return
			}

			name := ques.Name[:n+1]
			if clnt.DnsSuffix != "" {
				name += clnt.DnsSuffix + "."
			}

			origNames[i] = ques.Name
			req.Question[i].Name = name
		}

		defer func() {
			for i, _ := range req.Question {
				req.Question[i].Name = origNames[i]
			}
		}()

		for i, svr := range servers {
			if strings.Index(svr, ":") == -1 {
				servers[i] = svr + ":53"
			}
		}

		msg, err = r.Lookup(proto, servers, req)
		if err != nil {
			return
		}

		for i, _ := range msg.Question {
			msg.Question[i].Name = origNames[i]
		}
	}

	return
}

func (r *Resolver) LookupReverse(ques *question.Question, req *dns.Msg) (
	msg *dns.Msg, err error) {

	if ques.Qtype != dns.TypePTR {
		err = &NotFoundError{
			errors.New("resolver: Invalid dns type"),
		}
		return
	}

	db := database.GetDatabase()
	defer db.Close()
	coll := db.Clients()

	clnt := Client{}
	err = coll.Find(bson.M{
		"virt_address_num": ques.AddressNum,
	}).Select(bson.M{
		"domain_name": 1,
	}).One(&clnt)
	if err != nil {
		println(err.Error())
		err = database.ParseError(err)
		return
	}

	msg = &dns.Msg{}
	msg.SetReply(req)

	header := dns.RR_Header{
		Name:   ques.Name,
		Rrtype: dns.TypePTR,
		Class:  dns.ClassINET,
		Ttl:    5,
	}
	record := &dns.PTR{
		Hdr: header,
		Ptr: clnt.DomainName + ".vpn.",
	}
	msg.Answer = make([]dns.RR, 1)
	msg.Answer[0] = record

	return
}

func (r *Resolver) Lookup(proto string, servers []string, req *dns.Msg) (
	res *dns.Msg, err error) {

	client := &dns.Client{
		Net:          proto,
		Timeout:      3 * time.Second,
		DialTimeout:  r.Timeout,
		ReadTimeout:  r.Timeout,
		WriteTimeout: r.Timeout,
	}

	resChan := make(chan *dns.Msg, 1)
	waiter := utils.WaitCancel{}
	var ticker *time.Ticker
	var resErr error

	if len(servers) == 0 {
		servers = r.DefaultServers
	}

	if len(servers) > 2 {
		ticker = time.NewTicker(r.Interval)
	}

	for i, nameserver := range servers {
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
