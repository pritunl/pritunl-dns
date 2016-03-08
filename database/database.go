package database

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/Sirupsen/logrus"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-dns/constants"
	"gopkg.in/mgo.v2"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	mongoUrl    string
	mongoPrefix string
	mongoRate   time.Duration
	Session     *mgo.Session
)

type Database struct {
	session  *mgo.Session
	database *mgo.Database
}

func (d *Database) Close() {
	d.session.Close()
}

func (d *Database) getCollection(name string) (coll *Collection) {
	coll = &Collection{
		*d.database.C(name),
		d,
	}
	return
}

func (d *Database) Clients() (coll *Collection) {
	coll = d.getCollection(mongoPrefix + "clients")
	return
}

func (d *Database) Servers() (coll *Collection) {
	coll = d.getCollection(mongoPrefix + "servers")
	return
}

func Connect() (err error) {
	mgoUrl, err := url.Parse(mongoUrl)
	if err != nil {
		err = &ConnectionError{
			errors.Wrap(err, "database: Failed to parse mongo uri"),
		}
		return
	}

	vals := mgoUrl.Query()
	mgoSsl := vals.Get("ssl")
	mgoSslCerts := vals.Get("ssl_ca_certs")
	vals.Del("ssl")
	vals.Del("ssl_ca_certs")
	mgoUrl.RawQuery = vals.Encode()
	mgoUri := mgoUrl.String()

	if mgoSsl == "true" {
		info, e := mgo.ParseURL(mgoUri)
		if e != nil {
			err = &ConnectionError{
				errors.Wrap(e, "database: Failed to parse mongo url"),
			}
			return
		}

		info.DialServer = func(addr *mgo.ServerAddr) (
			conn net.Conn, err error) {

			tlsConf := &tls.Config{}

			if mgoSslCerts != "" {
				caData, e := ioutil.ReadFile(mgoSslCerts)
				if e != nil {
					err = &CertificateError{
						errors.Wrap(e, "database: Failed to load certificate"),
					}
					return
				}

				caPool := x509.NewCertPool()
				if ok := caPool.AppendCertsFromPEM(caData); !ok {
					err = &CertificateError{
						errors.Wrap(err,
							"database: Failed to parse certificate"),
					}
					return
				}

				tlsConf.RootCAs = caPool
			}

			conn, err = tls.Dial("tcp", addr.String(), tlsConf)
			return
		}
		Session, err = mgo.DialWithInfo(info)
		if err != nil {
			err = &ConnectionError{
				errors.Wrap(err, "database: Connection error"),
			}
			return
		}
	} else {
		Session, err = mgo.Dial(mgoUri)
		if err != nil {
			err = &ConnectionError{
				errors.Wrap(err, "database: Connection error"),
			}
			return
		}
	}

	Session.SetMode(mgo.Strong, true)

	return
}

func GetDatabase() (db *Database) {
	session := Session.Copy()

	var dbName string
	if x := strings.LastIndex(mongoUrl, "/"); x != -1 {
		dbName = mongoUrl[x+1:]
	} else {
		dbName = "pritunl"
	}

	database := session.DB(dbName)

	db = &Database{
		session:  session,
		database: database,
	}
	return
}

func init() {
	mongoUrl = os.Getenv("DB")
	mongoPrefix = os.Getenv("DB_PREFIX")

	mongoRateStr := os.Getenv("DB_SYNC_RATE")
	if mongoRateStr != "" {
		mongoRateNum, err := strconv.Atoi(mongoRateStr)
		if err != nil {
			panic(err)
		}

		mongoRate = time.Duration(mongoRateNum) * time.Second
	} else {
		mongoRate = constants.DefaultDatabaseSyncRate
	}

	for {
		err := Connect()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Error("database: Connection")
		} else {
			break
		}

		time.Sleep(1 * time.Second)
	}

	go dnsSync()
}
