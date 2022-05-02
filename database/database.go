package database

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/mongo-go-driver/mongo"
	"github.com/pritunl/mongo-go-driver/mongo/options"
	"github.com/pritunl/mongo-go-driver/x/mongo/driver/connstring"
	"github.com/pritunl/pritunl-dns/constants"
	"github.com/sirupsen/logrus"
)

var (
	mongoUri        string
	mongoPrefix     string
	mongoRate       time.Duration
	Client          *mongo.Client
	DefaultDatabase string
)

type Database struct {
	ctx      context.Context
	client   *mongo.Client
	database *mongo.Database
}

func (d *Database) Deadline() (time.Time, bool) {
	if d.ctx != nil {
		return d.ctx.Deadline()
	}
	return time.Time{}, false
}

func (d *Database) Done() <-chan struct{} {
	if d.ctx != nil {
		return d.ctx.Done()
	}
	return nil
}

func (d *Database) Err() error {
	if d.ctx != nil {
		return d.ctx.Err()
	}
	return nil
}

func (d *Database) Value(key interface{}) interface{} {
	if d.ctx != nil {
		return d.ctx.Value(key)
	}
	return nil
}

func (d *Database) String() string {
	return "context.database"
}

func (d *Database) Close() {
}

func (d *Database) getCollection(name string) (coll *Collection) {
	coll = &Collection{
		db:         d,
		Collection: d.database.Collection(name),
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
	mongoUrl, err := connstring.ParseAndValidate(mongoUri)
	if err != nil {
		err = &ConnectionError{
			errors.Wrap(err, "database: Failed to parse mongo uri"),
		}
		return
	}

	if mongoUrl.Database != "" {
		DefaultDatabase = mongoUrl.Database
	}

	opts := options.Client().ApplyURI(mongoUri)
	client, err := mongo.NewClient(opts)
	if err != nil {
		err = &ConnectionError{
			errors.Wrap(err, "database: Client error"),
		}
		return
	}

	err = client.Connect(context.TODO())
	if err != nil {
		err = &ConnectionError{
			errors.Wrap(err, "database: Connection error"),
		}
		return
	}

	Client = client

	return
}

func GetDatabase() (db *Database) {
	client := Client
	if client == nil {
		return
	}

	database := client.Database(DefaultDatabase)

	db = &Database{
		client:   client,
		database: database,
	}
	return
}

func GetDatabaseCtx(ctx context.Context) (db *Database) {
	client := Client
	if client == nil {
		return
	}

	database := client.Database(DefaultDatabase)

	db = &Database{
		ctx:      ctx,
		client:   client,
		database: database,
	}
	return
}

func Init() {
	mongoUri = os.Getenv("DB")
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
			}).Error("database: Connection error")
		} else {
			break
		}

		time.Sleep(1 * time.Second)
	}

	go dnsSync()

	return
}
