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

	if dns.IsFqdn(q.Name) {
		q.NameTrim = q.Name[:len(q.Name)-1]
	} else {
		q.NameTrim = q.Name
	}

	name := q.NameTrim

	n := strings.LastIndex(name, ".")
	if n == -1 {
		return
	}

	q.TopDomain = name[n+1:]
	q.Domain = name[:n]

	if q.Qtype == dns.TypePTR {
		addr := strings.Replace(q.Name, ".in-addr.arpa.", "", 1)
		addrSpl := strings.Split(addr, ".")

		for i, x := range addrSpl {
			y, _ := strconv.ParseInt(x, 10, 64)
			q.AddressNum += y << uint(8*i)
		}
	}

	return
}
