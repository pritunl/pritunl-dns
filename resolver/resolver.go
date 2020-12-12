package resolver

import (
	"crypto/md5"
	"net"
	"strings"
	"time"

	"github.com/dropbox/godropbox/errors"
	"github.com/miekg/dns"
	"github.com/pritunl/pritunl-dns/database"
	"github.com/pritunl/pritunl-dns/question"
	"github.com/pritunl/pritunl-dns/utils"
	"gopkg.in/mgo.v2/bson"
)

type Client struct {
	Network      string   `bson:"network"`
	NetworkWg    string   `bson:"network_wg"`
	DomainName   string   `bson:"domain_name"`
	VirtAddress  string   `bson:"virt_address"`
	VirtAddress6 string   `bson:"virt_address6"`
	DnsServers   []string `bson:"dns_servers"`
	DnsSuffix    string   `bson:"dns_suffix"`
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
		"network":       1,
		"network_wg":    1,
		"virt_address":  1,
		"virt_address6": 1,
		"dns_servers":   1,
		"dns_suffix":    1,
	}).Iter()

	clnt := Client{}
	for cursor.Next(&clnt) {
		if clnt.Network == subnet || clnt.NetworkWg == subnet ||
			subnet == "" {

			break
		}
	}

	err = cursor.Close()
	if err != nil {
		err = database.ParseError(err)
		return
	}

	if subDomain == "" {
		msg = &dns.Msg{}
		msg.SetReply(req)
		msg.Answer = make([]dns.RR, 0)

		if ques.Qtype == dns.TypeA {
			record, e := r.createIpv4Record(ques, clnt)
			if e != nil {
				err = &ResolveError{
					errors.Wrap(e, "resolver: Failed to create record"),
				}
				return
			}

			msg.Answer = append(msg.Answer, record)
		}

		if ques.Qtype == dns.TypeAAAA && clnt.VirtAddress6 != "" {
			record6, e := r.createIpv6Record(ques, clnt)
			if e != nil {
				err = &ResolveError{
					errors.Wrap(e, "resolver: Failed to create record6"),
				}
				return
			}

			msg.Answer = append(msg.Answer, record6)
		}
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

		for i := range msg.Question {
			msg.Question[i].Name = origNames[i]
		}
	}

	return
}

func (r *Resolver) createIpv4Record(ques *question.Question,
	clnt Client) (record dns.RR, err error) {

	clientIpStr := strings.Split(clnt.VirtAddress, "/")[0]
	if clientIpStr == "" {
		err = &UnknownError{
			errors.New("resolver: Failed to find IPv4"),
		}
		return
	}

	clientIp := net.ParseIP(clientIpStr)
	if clientIp == nil {
		err = &UnknownError{
			errors.New("resolver: Unknown parse IPv4 error"),
		}
		return
	}

	header := dns.RR_Header{
		Name:   ques.Name,
		Rrtype: dns.TypeA,
		Class:  dns.ClassINET,
		Ttl:    5,
	}
	record = &dns.A{
		Hdr: header,
		A:   clientIp,
	}

	return
}

func (r *Resolver) createIpv6Record(ques *question.Question,
	clnt Client) (record6 dns.RR, err error) {

	clientIpStr6 := strings.Split(clnt.VirtAddress6, "/")[0]
	if clientIpStr6 == "" {
		err = &UnknownError{
			errors.New("resolver: Failed to find IPv6"),
		}
		return
	}

	clientIp6 := net.ParseIP(clientIpStr6)
	if clientIp6 == nil {
		err = &UnknownError{
			errors.New("resolver: Unknown parse IPv6 error"),
		}
		return
	}

	header6 := dns.RR_Header{
		Name:   ques.Name,
		Rrtype: dns.TypeAAAA,
		Class:  dns.ClassINET,
		Ttl:    5,
	}
	record6 = &dns.AAAA{
		Hdr:  header6,
		AAAA: clientIp6,
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

	if len(servers) == 0 {
		servers = r.DefaultServers
	}

	for _, nameserver := range servers {
		res, _, err = client.Exchange(req, nameserver)
		if err != nil {
			err = &ResolveError{
				errors.Wrap(err, "resolver: Socket error"),
			}
			continue
		}

		break
	}

	if err != nil {
		return
	}

	return
}
