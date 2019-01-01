package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type spjs struct {
	Meta struct {
		Hostname string
		Version  string
	}

	url string

	mx          sync.RWMutex
	serialPorts []spjsSerialPort

	outgoing  chan spjsMessage
	incomming chan interface{}
}

type spjsMessage struct {
	done    chan struct{}
	payload []byte
}

type spjsMsgVersion struct {
	Version string
}
type spjsMsgCommands struct {
	Commands []string
}
type spjsMsgHostname struct {
	Hostname string
}
type spjsDataFrame struct {
	Port string `json:"P"`
	Data string `json:"D"`
}
type spjsCmdStatus struct {
	Cmd        string
	QueueCount int    `json:"QCnt"`
	Data       string `json:"D"`
	ID         string `json:"Id"`
}
type spjsCmdQStatus struct {
	Cmd        string
	QueueCount int `json:"QCnt"`
	Type       []string
	Data       []string `json:"D"`
	ID         string   `json:"Id"`
}

type spjsError struct {
	Error string
}
type spsjsPorts struct {
	SerialPorts []spjsSerialPort
}
type spjsSerialPort struct {
	Name                      string
	Friendly                  string
	SerialNumber              string
	DeviceClass               string
	IsOpen                    bool
	IsPrimary                 bool
	RelatedNames              []string
	Baud                      int
	BufferAlgorithm           string
	AvailableBufferAlgorithms []string
	Ver                       float64
	USBVID                    string
	USBPID                    string
	FeedRateOverride          float64
}

func newSPJS(url string) *spjs {
	sp := &spjs{
		url:       url,
		outgoing:  make(chan spjsMessage, 1000),
		incomming: make(chan interface{}, 1000),
	}

	go sp.loop()

	return sp
}

func parseSPJSMessage(data []byte, msg map[string]json.RawMessage) (val interface{}, err error) {
	check := func(fieldName string, v interface{}) bool {
		if msg[fieldName] == nil {
			return false
		}
		val = v
		err = json.Unmarshal(data, val)
		return true
	}
	if check("Hostname", &spjsMsgHostname{}) {
		return
	}
	if check("Commands", &spjsMsgCommands{}) {
		return
	}
	if check("Version", &spjsMsgVersion{}) {
		return
	}
	if check("SerialPorts", &spsjsPorts{}) {
		return
	}
	if check("Type", &spjsCmdQStatus{}) {
		return
	}
	if check("Cmd", &spjsCmdQStatus{}) {
		return
	}
	if check("D", &spjsDataFrame{}) {
		return
	}

	return nil, errors.New("unknown message: " + string(data))
}
func (sp *spjs) readLoop(ws *websocket.Conn, done chan struct{}) {
	defer close(done)
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			log.Println("ERROR: read:", err)
			return
		}
		if !bytes.HasPrefix(data, []byte("{")) {
			// ignore echo messages
			continue
		}
		var msg map[string]json.RawMessage
		err = json.Unmarshal(data, &msg)
		if err != nil {
			log.Println("ERROR: read:", err)
		}
		val, err := parseSPJSMessage(data, msg)
		if err != nil {
			log.Println("ERROR: parse:", err)
			continue
		}
		sp.incomming <- val
	}
}
func (sp *spjs) loop() {
	var nextUp spjsMessage

reconnect:
	for {
		log.Println("Connecting to", sp.url)
		ws, _, err := websocket.DefaultDialer.Dial(sp.url, nil)
		if err != nil {
			log.Println("ERROR: connect:", err)
			time.Sleep(3 * time.Second)
			continue
		}
		log.Println("Connected.")
		ch := make(chan struct{})
		go sp.readLoop(ws, ch)
		go sp.WriteString("list") // always get list first

		for {
			if nextUp.done != nil {
				err = ws.WriteMessage(websocket.TextMessage, nextUp.payload)
				if err != nil {
					log.Println("ERROR: send:", err)
					continue reconnect
				}
				close(nextUp.done)
				nextUp.done = nil
			}

			select {
			case <-ch:
				continue reconnect
			case nextUp = <-sp.outgoing:
			}
		}
	}
}

type spjsJSON struct {
	Port string `json:"P"`
	Data []spjsData
}
type spjsData struct {
	Data string `json:"D"`
	ID   string `json:"Id"`
}

func (sp *spjs) SendJSON(v spjsJSON) {
	data, err := json.Marshal(v)
	if err != nil {
		// shouldn't happen since we control everything that's sent out
		log.Panicln("ERROR: sendjson (marshal):", err)
		return
	}

	ch := make(chan struct{})
	sp.outgoing <- spjsMessage{done: ch, payload: append([]byte("sendjson "), data...)}
	<-ch
}
func (sp *spjs) WriteString(data string) {
	ch := make(chan struct{})
	sp.outgoing <- spjsMessage{done: ch, payload: []byte(data)}
	<-ch
}
