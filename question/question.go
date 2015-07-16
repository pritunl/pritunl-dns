package question

import (
	"github.com/miekg/dns"
	"strings"
)

type Question struct {
	dns.Question
	NameTrim  string
	TopDomain string
	Domain    string
	IsIpQuery bool
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

	return
}
