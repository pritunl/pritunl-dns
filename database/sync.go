package database

import (
	"fmt"
	"labix.org/v2/mgo/bson"
	"time"
)

var DnsServers = map[string][]string{}

type server struct {
	Network    string   `bson:"network"`
	DnsServers []string `bson:"dns_servers"`
}

func dnsSync() {
	for {
		time.Sleep(mongoRate)

		dnsServers := map[string][]string{}

		db := GetDatabase()
		coll := db.Servers()

		cursor := coll.Find(bson.M{}).Iter()

		svr := server{}
		for cursor.Next(&svr) {
			for i, dnsSvr := range svr.DnsServers {
				svr.DnsServers[i] = dnsSvr + ":53"
			}

			dnsServers[svr.Network] = svr.DnsServers
		}

		DnsServers = dnsServers

		fmt.Println(DnsServers)
	}
}
