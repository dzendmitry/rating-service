package mongo

import (
	"github.com/dzendmitry/logger"
	"time"
	"gopkg.in/mgo.v2"
	"fmt"
	"strings"
)

var (
	dbURI string
	dbName string
	dbAuthUser string
	dbAuthPass string
	session *mgo.Session
	db *mgo.Database
	log logger.ILogger
	dbStateCh       = make(chan chan bool, 30)
	dbConnectCh     = make(chan int, 1)
	dbDisconnectCh  = make(chan int, 30)
	stopReconnectCh = make(chan int, 1)
)

const (
	DIAL_TIMEOUT   = 30 * time.Second
	SYNC_TIMEOUT   = 1 * time.Minute
	SOCKET_TIMEOUT = 1 * time.Minute
)

func connect() error {
	log.Infof("dbUri: %s", dbURI)

	s, err := mgo.DialWithTimeout(dbURI, DIAL_TIMEOUT)
	if err != nil {
		log.Fatalf("Error connecting to db: %v", err)
		return err
	}
	session = s
	session.SetSyncTimeout(SYNC_TIMEOUT)
	session.SetSocketTimeout(SOCKET_TIMEOUT)

	log.Infof("dbName: %s", dbName)
	db = session.DB(dbName)
	if err != nil {
		log.Fatalf("Error connecting to db: %v", err)
		session.Close()
		return err
	}
	if dbAuthUser != "" || dbAuthPass != "" {
		if err = db.Login(dbAuthUser, dbAuthPass); err != nil {
			log.Fatalf("Error login to db: %v", err)
			session.Close()
			return err
		}
	}
	return nil
}

func Start(uri, db, user, pass string) error {
	log = logger.InitFileLogger("MONGO", "")

	dbURI = uri
	dbName = db
	dbAuthUser = user
	dbAuthPass = pass

	err := connect()
	if err != nil {
		return err
	}

	go connectionKeeper()

	return nil
}

func connectionKeeper() {
	isActive := true
L:
	for {
		select {
		case ch, ok := <-dbStateCh:
			if !ok {
				break L
			}
			ch <- isActive
		case <-dbDisconnectCh:
			if isActive {
				log.Fatalf("*** connection is lost! ***")
				isActive = false
				go reconnectDB()
			}
		case <-dbConnectCh:
			isActive = true
			log.Fatalf("connection is restored!")
		}
	}
}

func reconnectDB() {
	reconnectedCh := make(chan int, 1)
	go func() {
		for i := 0; ; i++ {
			msg := ""
			session.Refresh()

			if len(session.LiveServers()) == 0 {
				msg = "No live servers"
			} else {
				s := GetSessionCopy()
				_, err := s.collection("local").Find(nil).Count()
				s.Close()
				if err != nil {
					msg = fmt.Sprintf("Error making query to DB: %v", err)
				}
			}

			if msg == "" {
				reconnectedCh <- 1
				return
			}
			if i%5 == 0 {
				log.Fatalf(msg)
			}
			time.Sleep(time.Second)
		}
	}()

	select {
	case <-stopReconnectCh:
		return
	case <-reconnectedCh:
		dbConnectCh <- 1
		return
	}
}

func isActive() bool {
	ch := make(chan bool, 1)
	dbStateCh <- ch
	return <-ch
}

func Close() {
	log.Info("Close db")
	stopReconnectCh <- 1
	close(dbStateCh)
	session.Close()
}

func disconnectDetected() {
	dbDisconnectCh <- 1
}

func worthRefresh(err error) bool {
	msg := strings.ToLower(err.Error())
	return err != nil && (msg == "closed explicitly" || msg == "eof" || msg == "no reachable servers")
}

type Session struct {
	*mgo.Session
}

func GetSessionCopy() *Session {
	return &Session{session.Copy()}
}

func (s *Session) collection(cName string) *mgo.Collection {
	return s.DB(dbName).C(cName)
}

func (s *Session) Find(cName string, query interface{}) *mgo.Query {
	log.Infof("Executing in %s query %#v", cName, query)

	mgoQuery := s.collection(cName).Find(query)

	_, err := mgoQuery.Count()
	if err != nil {
		log.Infof("error executing query: %s (%v)", err.Error(), query)
		if worthRefresh(err) {
			s.Refresh()
			mgoQuery = s.collection(cName).Find(query)
			_, err = mgoQuery.Count()
			if err != nil {
				log.Fatalf("retry attempt: error executing query: %s (%v)", err.Error(), query)
				disconnectDetected()
			}
		}
	}
	return mgoQuery
}

func (s *Session) Insert(cName string, docs ...interface{}) error {
	log.Infof("inserting to %s documents %#v", cName, docs)

	c := s.collection(cName)

	err := c.Insert(docs...)
	if err != nil {
		log.Infof("error inserting to %s: %s (%v)", cName, err.Error(), docs)
		if worthRefresh(err) {
			s.Refresh()
			err = c.Insert(docs...)
			if err != nil {
				log.Fatalf("retry attempt: error inserting to %s: %s (%v)", cName, err.Error(), docs)
				disconnectDetected()
			}
		}
	}
	return err
}

func (s *Session) Remove(cName string, selector interface{}) error {
	log.Infof("removing document from %s %#v", cName, selector)

	c := s.collection(cName)

	err := c.Remove(selector)
	if err != nil {
		log.Infof("error removing document from %s: %s (%#v)", cName, err.Error(), selector)
		if worthRefresh(err) {
			s.Refresh()
			err = c.Remove(selector)
			if err != nil {
				log.Fatalf("retry attempt: error removing document from %s: %s (%#v)", cName, err.Error(), selector)
				disconnectDetected()
			}
		}
	}
	return err
}

func (s *Session) Update(cName string, selector, update interface{}) error {
	log.Infof("updating document in %s %#v with %#v", cName, selector, update)

	c := s.collection(cName)

	err := c.Update(selector, update)
	if err != nil {
		log.Infof("error updating document in %s: %s (%#v, %#v)", cName, err.Error(), selector, update)
		if worthRefresh(err) {
			s.Refresh()
			err = c.Update(selector, update)
			if err != nil {
				log.Fatalf("retry attempt: error updating document in %s: %s (%#v, %#v)", cName, err.Error(), selector, update)
				disconnectDetected()
			}
		}
	}
	return err
}

func (s *Session) Upsert(cName string, selector, update interface{}) error {
	log.Infof("upserting document into %s %#v with %#v", cName, selector, update)

	c := s.collection(cName)

	_, err := c.Upsert(selector, update)
	if err != nil {
		log.Infof("error upserting document into %s: %s (%#v, %#v)", cName, err.Error(), selector, update)
		if worthRefresh(err) {
			s.Refresh()
			_, err = c.Upsert(selector, update)
			if err != nil {
				log.Fatalf("retry attempt: error upserting document into %s: %s (%#v, %#v)", cName, err.Error(), selector, update)
				disconnectDetected()
			}
		}
	}
	return err
}