package mdb

import (
	"github.com/micro/go-log"
	"time"
	"gopkg.in/mgo.v2"

	"github.com/lakstap/go-atk/database/config"
)

type DatabaseSession struct {
	*mgo.Session
	databaseName string
}

func GetDBSession(DBConfig config.DBConfig) (*DatabaseSession, error) {
	// establish mongo db session.
	mongoDBDialInfo := &mgo.DialInfo{
		Addrs:    []string{DBConfig.Address},
		Timeout:  60 * time.Second,
		Database: DBConfig.Database,
		Username: DBConfig.UserName,
		Password: DBConfig.Password,
	}

	// Create a session which maintains a pool of socket connections
	// to our MongoDB.
	mongoSession, err := mgo.DialWithInfo(mongoDBDialInfo)
	if err != nil {
		log.Fatalf("Database Session Creation ERROR: %s\n", err)
	}
	mongoSession.SetMode(mgo.Monotonic, true)
	return &DatabaseSession{mongoSession, DBConfig.Database}, err
}
