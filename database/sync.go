package database

import (
	"gopkg.in/mgo.v2/bson"
	"time"
)

var DnsServers = map[string][]string{}

type server struct {
	Network    string   `bson:"network"`
	DnsServers []string `bson:"dns_servers"`
}

func sync() (err error) {
	dnsServers := map[string][]string{}

	db := GetDatabase()
	defer db.Close()
	coll := db.Servers()

	cursor := coll.Find(bson.M{}).Select(bson.M{
		"network":     1,
		"dns_servers": 1,
	}).Iter()

	svr := server{}
	for cursor.Next(&svr) {
		for i, dnsSvr := range svr.DnsServers {
			svr.DnsServers[i] = dnsSvr + ":53"
		}

		dnsServers[svr.Network] = svr.DnsServers
	}

	err = cursor.Close()
	if err != nil {
		err = ParseError(err)
		return
	}

	DnsServers = dnsServers

	return
}

func dnsSync() {
	for {
		sync()
		time.Sleep(mongoRate)
	}
}
