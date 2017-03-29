package udp

import (
	"net"
	"time"
	"encoding/gob"
	"bytes"
	"github.com/dzendmitry/logger"
	"log"
)

const TICKER_TIMEOUT = 500

type ISender interface {
	Send()
}

type ParsersSender struct {
	log logger.ILogger
	addr *net.UDPAddr
	connected bool
	data []byte
}

func NewParsersSender(host, port string, m ParserMessage, log logger.ILogger) *ParsersSender {
	addr, err := net.ResolveUDPAddr("udp", host + ":" + port)
	if err != nil {
		log.Panicf("Resolving udp addr error: %+v", err.Error())
		panic(err)
	}
	var data bytes.Buffer
	enc := gob.NewEncoder(&data)
	if err := enc.Encode(m); err != nil {
		log.Panicf("Encoding parser message to binary buffer error: %+v", err.Error())
		panic(err)
	}
	return &ParsersSender{
		log: log,
		addr: addr,
		data: data.Bytes(),
	}
}

func (p *ParsersSender) Send() {
	go func() {
		var c *net.UDPConn
		ticker := time.NewTicker(TICKER_TIMEOUT * time.Millisecond)
		for {
			<-ticker.C
			if !p.connected {
				var err error
				c, err = net.DialUDP("udp", nil, p.addr)
				if err != nil {
					log.Fatalf("Error while dialing udp addr: %+v", err.Error())
				} else {
					p.connected = true
				}
			}
			if p.connected {
				n, err := c.Write(p.data)
				if err != nil {
					log.Fatalf("Error writing to udp socker: %+v", err.Error())
				} else if n < len(p.data) {
					log.Fatalf("Error writing to udp socket message len: %d", n)
				}
			}
		}
	}()
}