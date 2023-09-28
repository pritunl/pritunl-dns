package database

import (
	"time"

	"github.com/pritunl/mongo-go-driver/bson"
	"github.com/pritunl/mongo-go-driver/mongo/options"
	"github.com/sirupsen/logrus"
)

var DnsServers = map[string][]string{}

type server struct {
	Network    string   `bson:"network"`
	NetworkWg  string   `bson:"network_wg"`
	DnsServers []string `bson:"dns_servers"`
}

func sync() (err error) {
	dnsServers := map[string][]string{}

	db := GetDatabase()
	defer db.Close()
	coll := db.Servers()

	cursor, err := coll.Find(
		db,
		&bson.M{},
		&options.FindOptions{
			Projection: &bson.D{
				{"network", 1},
				{"network_wg", 1},
				{"dns_servers", 1},
			},
		},
	)
	if err != nil {
		err = ParseError(err)
		return
	}
	defer cursor.Close(db)

	for cursor.Next(db) {
		svr := &server{}
		err = cursor.Decode(svr)
		if err != nil {
			err = ParseError(err)
			return
		}

		for i, dnsSvr := range svr.DnsServers {
			svr.DnsServers[i] = dnsSvr + ":53"
		}

		dnsServers[svr.Network] = svr.DnsServers
		dnsServers[svr.NetworkWg] = svr.DnsServers
	}

	err = cursor.Err()
	if err != nil {
		err = ParseError(err)
		return
	}

	DnsServers = dnsServers

	return
}

func dnsSync() {
	for {
		err := sync()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Error("database: Sync dns error")
		}

		time.Sleep(mongoRate)
	}
}
