package redis

import (
	"errors"
	"fmt"
	"time"
	"unsafe"
	"sync/atomic"

	"github.com/mediocregopher/radix.v2/sentinel"
	"github.com/mediocregopher/radix.v2/redis"
	"github.com/dzendmitry/logger"
)

const (
	POOL_SIZE          = 400
	CACHE_EVICT        = "redis-cache-evict"
	CACHE_EVICT_EX     = 86400
	GET_MASTER_TIMEOUT = 100
	DIAL_TIMEOUT       = 1000
)

var (
	sentielNames []string
	masterNames []string
	client *sentinel.Client
	log logger.ILogger
	dbStateCh       = make(chan chan bool, 30)
	dbConnectCh     = make(chan int, 1)
	dbDisconnectCh  = make(chan int, 30)
	stopReconnectCh = make(chan int, 1)
)

func dial(network, addr string) (*redis.Client, error) {
	return redis.DialTimeout(network, addr, time.Duration(DIAL_TIMEOUT * time.Millisecond))
}

func getClient() *sentinel.Client {
	cas := false
	var c *sentinel.Client
	for !cas {
		c = client
		if c == nil {
			return nil
		}
		cas = atomic.CompareAndSwapPointer(
			(*unsafe.Pointer)(unsafe.Pointer(&client)),
			unsafe.Pointer(c),
			unsafe.Pointer(c),
		)
	}
	return c
}

func Start(sentiels, masters []string) error {
	log = logger.InitFileLogger("REDIS", "")
	sentielNames = sentiels
	masterNames = masters
	if err := connect(); err != nil {
		return err
	}
	go connectionKeeper()
	return nil
}

func Close() {
	log.Info("Close db")
	stopReconnectCh <- 1
	close(dbStateCh)
	log.Close()
}

type connectResp struct {
	client *sentinel.Client
	err error
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
				go reconnect()
			}
		case <-dbConnectCh:
			isActive = true
			log.Fatalf("connection is restored!")
		}
	}
}

func connect() error {
	if len(sentielNames) == 0 {
		return errors.New("There is no one sentiel in the list")
	}
	errs := make([]string, 0, len(sentielNames))
	for _, sentiel := range sentielNames {
		c, err := sentinel.NewClientCustom("tcp", sentiel, POOL_SIZE, dial, masterNames...)
		if err != nil {
			errs = append(errs, sentiel + " " + err.Error())
		} else {
			atomic.SwapPointer((*unsafe.Pointer)(unsafe.Pointer(&client)), unsafe.Pointer(c))
			return nil
		}
	}
	return errors.New(fmt.Sprintf("There is no one redis master or all sentinels are down: %#v", errs))
}

func reconnect() {
	reconnectedCh := make(chan int, 1)
	go func() {
		for i := 0; ; i++ {
			err := connect()
			if err != nil {
				time.Sleep(time.Second)
			} else {
				reconnectedCh <- 1
				return
			}
			if err != nil && i%5 == 0 {
				log.Fatalf(err.Error())
			}
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

func disconnectDetected() {
	dbDisconnectCh <- 1
}

func SetExSentiel(master, key string, data []byte, ex int) error {
	if len(data) == 0 {
		return errors.New("There is no data for writing to redis")
	}
	var conn *redis.Client
	var err error
	c := getClient()
	if c == nil {
		disconnectDetected()
		return errors.New("Redis cache disconnected")
	}
	if conn, err = c.GetMaster(master); err != nil {
		disconnectDetected()
		time.Sleep(GET_MASTER_TIMEOUT * time.Millisecond)
		if conn, err = c.GetMaster(master); err != nil {
			return err
		}
	}
	if err = conn.Cmd("SET", key, string(data), "EX", ex).Err; err != nil {
		return err
	}
	c.PutMaster(master, conn)
	return nil
}

func GetSentiel(master, key string) ([]byte, error) {
	var conn *redis.Client
	var err error
	c := getClient()
	if c == nil {
		disconnectDetected()
		return nil, errors.New("Redis cache disconnected")
	}
	if conn, err = c.GetMaster(master); err != nil {
		disconnectDetected()
		time.Sleep(GET_MASTER_TIMEOUT * time.Millisecond)
		if conn, err = c.GetMaster(master); err != nil {
			return nil, err
		}
	}
	var str string
	str, err = conn.Cmd("GET", key).Str()
	if err != nil {
		return nil, err
	}
	c.PutMaster(master, conn)
	return []byte(str), nil
}