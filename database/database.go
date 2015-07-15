package database

import (
	"github.com/Sirupsen/logrus"
	"github.com/blckur/blckur/constants"
	"github.com/blckur/blckur/requires"
	"github.com/dropbox/godropbox/errors"
	"labix.org/v2/mgo"
	"os"
	"strings"
	"time"
)

var (
	mongoUrl    string
	mongoPrefix string
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

func (d *Database) UsersIp() (coll *Collection) {
	coll = d.getCollection(mongoPrefix + "users_ip")
	return
}

func Connect() (err error) {
	Session, err = mgo.Dial(mongoUrl)
	if err != nil {
		err = &ConnectionError{
			errors.Wrap(err, "database: Connection error"),
		}
		return
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

	module := requires.New("database")

	module.Handler = func() {
		for {
			err := Connect()
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"error": err,
				}).Error("database: Connection")
			} else {
				break
			}

			time.Sleep(constants.RetryDelay)
		}
	}
}
