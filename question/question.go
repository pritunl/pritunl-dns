package question

import (
	"github.com/miekg/dns"
	"strconv"
	"strings"
)

type Question struct {
	dns.Question
	NameTrim   string
	TopDomain  string
	Domain     string
	AddressNum int64
	IsIpQuery  bool
}

func (q *Question) isIpQuery() bool {
	if q.Qclass != dns.ClassINET {
		return false
	}

	switch q.Qtype {
	case dns.TypeA, dns.TypeAAAA:
		return true
	default:
		return false
	}
}

func NewQuestion(ques dns.Question) (q *Question) {
	q = &Question{
		Question: ques,
	}

	q.IsIpQuery = q.isIpQuery()

	name := strings.ToLower(q.Name)

	if dns.IsFqdn(name) {
		q.NameTrim = name[:len(name)-1]
	} else {
		q.NameTrim = name
	}

	n := strings.LastIndex(q.NameTrim, ".")
	if n == -1 {
		return
	}

	q.TopDomain = q.NameTrim[n+1:]
	q.Domain = q.NameTrim[:n]

	if q.Qtype == dns.TypePTR {
		addr := strings.Replace(name, ".in-addr.arpa.", "", 1)
		addrSpl := strings.Split(addr, ".")

		for i, x := range addrSpl {
			y, _ := strconv.ParseInt(x, 10, 64)
			q.AddressNum += y << uint(8*i)
		}
	}

	return
}
