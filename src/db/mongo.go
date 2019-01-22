package db

import (
	"os"

	"github.com/globalsign/mgo"
)

func GetMongoSession() (*mgo.Session, error) {
	env := os.Getenv("ENVIRONMENT")
	var url string

	if env == "PRODUCTION" {
		url = "mongodb://api:dM6CYayNQu8qr9b@ds149984.mlab.com:49984/heroku_rjnls62m"
	} else {
		url = "localhost"
	}
	session, err := mgo.Dial(url)
	if err != nil {
		return nil, err
	}
	session.SetSafe(&mgo.Safe{})
	return session, nil
}

func CloseMongoSession(session *mgo.Session) {
	session.Close()
}
