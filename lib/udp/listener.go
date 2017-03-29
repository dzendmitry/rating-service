package udp

import (
	"time"
	"net"
	"errors"
	"encoding/gob"
	"bytes"

	"github.com/dzendmitry/rating-service/lib/general"
	"github.com/dzendmitry/logger"
	"log"
)

const (
	PL_UDP_MESSAGES_CHAN_LEN = 1000
	PL_TYPE_CHAN_LEN = 100000
	PL_WAITING_MESSAGE_TIMEOUT = 500
	PL_UDP_MESSAGE_BUFFER_SIZE = 1024
	PL_PARSER_LIVETIME = 2
)

type UdpMessage struct {
	n int
	addr net.Addr
	data []byte
	err error
}

type ParserMessage struct {
	Name string
	ParserType string
	Hash string
	HttpHost string
	HttpPort string
}

type ParserUnit struct {
	ParserMessage
	lastUpdate time.Time
	Addr net.Addr
}

type GetParsersCmd struct {
	Type string
	C chan []ParserUnit
}

type ParsersListener struct {
	log logger.ILogger
	parsers []ParserUnit
	typeC chan GetParsersCmd
	addr *net.UDPAddr
	ifi *net.Interface
}

func NewParsersListener(host, port, ifis string, log logger.ILogger) *ParsersListener {
	addr, err := net.ResolveUDPAddr("udp", host + ":" + port)
	if err != nil {
		log.Panicf("Resolving udp addr error: %+v", err.Error())
		panic(err)
	}
	ifi, err := net.InterfaceByName(ifis)
	if err != nil {
		log.Panicf("Getting interface by name error: %+v", err.Error())
		panic(err)
	}
	return &ParsersListener{
		log: log,
		parsers: make([]ParserUnit, 0),
		typeC: make(chan GetParsersCmd, PL_TYPE_CHAN_LEN),
		addr: addr,
		ifi: ifi,
	}
}

func (pl *ParsersListener) GetTypeC() chan GetParsersCmd {
	return pl.typeC
}

func (pl *ParsersListener) Listen() {
	go func() {
		pc, err := net.ListenMulticastUDP("udp", pl.ifi, pl.addr)
		if err != nil {
			log.Panicf("Listening multicast udp error: %+v", err.Error())
			panic(err)
		}
		defer pc.Close()
		dataCh := make(chan *UdpMessage, PL_UDP_MESSAGES_CHAN_LEN)

		go func() {
			for {
				buffer := make([]byte, PL_UDP_MESSAGE_BUFFER_SIZE)
				n, addr, err := pc.ReadFrom(buffer)
				if err != nil {
					log.Fatalf("Reading from udp socket error: %+v", err.Error())
					dataCh <- &UdpMessage{err: err}
					continue
				}
				if n == 0 {
					log.Fatalf("Length of multicast udp packet is zero")
					dataCh <- &UdpMessage{err: errors.New("UDP message len is zero")}
					continue
				}
				dataCh <- &UdpMessage{n, addr, buffer, err}
			}
		}()

		ticker := time.NewTicker(PL_WAITING_MESSAGE_TIMEOUT * time.Millisecond)
		for {
			select {
			case message := <-dataCh:
				if message.err != nil {
					continue
				}
				pm := ParserMessage{}
				dec := gob.NewDecoder(bytes.NewReader(message.data))
				err = dec.Decode(&pm)
				if err != nil {
					log.Fatalf("Decoding multicast parser message error: %+v", err.Error())
					continue
				}
				modified := false
				for i, p := range pl.parsers {
					if p.ParserType == pm.ParserType && p.Name == pm.Name && p.Hash == pm.Hash {
						pl.parsers[i].lastUpdate = time.Now()
						modified = true
					}
				}
				if !modified {
					pl.parsers = append(pl.parsers, ParserUnit{
						ParserMessage: pm,
						lastUpdate: time.Now(),
						Addr: message.addr,
					})
					pl.log.Debugf("Parsed added: %+v", pl.parsers[len(pl.parsers)-1])
				}
			case pcmd := <- pl.typeC:
				pcmd.C <- pl.getParsersByType(pcmd.Type)
			case <-ticker.C:
				pl.removeOverdueParsers()
			}
		}
	}()
}

func (pl *ParsersListener) removeOverdueParsers() {
	for i, parser := range pl.parsers {
		if time.Since(parser.lastUpdate) >= PL_PARSER_LIVETIME * time.Second {
			pl.log.Debugf("Parser removed: %+v", parser)
			copy(pl.parsers[i:], pl.parsers[i+1:])
			pl.parsers[len(pl.parsers)-1] = ParserUnit{}
			pl.parsers = pl.parsers[:len(pl.parsers)-1]
		}
	}
}

func (pl *ParsersListener) getParsersByType(parserType string) []ParserUnit {
	if _, ok := general.ParserTypes[parserType]; !ok {
		return make([]ParserUnit, 0)
	}
	res := make([]ParserUnit, 0, len(pl.parsers))
	for _, parser := range pl.parsers {
		if parser.ParserType == parserType {
			res = append(res, parser)
		}
	}
	return res
}