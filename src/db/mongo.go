package db

import (
	"github.com/globalsign/mgo"
	"github.com/price-scrapper/src/config"
)

func GetMongoSession() (*mgo.Session, error) {
	session, err := mgo.Dial(config.GetURLDB())
	if err != nil {
		return nil, err
	}
	session.SetSafe(&mgo.Safe{})
	return session, nil
}

func CloseMongoSession(session *mgo.Session) {
	session.Close()
}
