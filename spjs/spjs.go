package spjs

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type SPJS struct {
	url string

	mx          sync.RWMutex
	serialPorts []SerialPort

	outgoing  chan message
	incomming chan interface{}
}

type message struct {
	done    chan struct{}
	payload []byte
}

type DataFrame struct {
	Port string `json:"P"`
	Data string `json:"D"`
}
type CmdStatus struct {
	Cmd        string
	QueueCount int `json:"QCnt"`
	Type       []string
	Data       []string `json:"D"`
	ID         string   `json:"Id"`
}

type ErrorMessage struct {
	Error string
}
type SerialPortList struct {
	SerialPorts []SerialPort
}
type SerialPort struct {
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

func NewSPJS(url string) *SPJS {
	sp := &SPJS{
		url:       url,
		outgoing:  make(chan message, 1000),
		incomming: make(chan interface{}, 1000),
	}

	go sp.loop()

	return sp
}
func (sp *SPJS) Messages() chan interface{} {
	return sp.incomming
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
	if check("Error", &ErrorMessage{}) {
		return
	}
	if check("SerialPorts", &SerialPortList{}) {
		return
	}
	if check("Type", &CmdStatus{}) {
		return
	}
	if check("D", &DataFrame{}) {
		return
	}

	return nil, errors.New("unknown message: " + string(data))
}
func (sp *SPJS) readLoop(ws *websocket.Conn, done chan struct{}) {
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
func (sp *SPJS) loop() {
	var nextUp message

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
		go sp.WriteString("list") // refresh list on reconnect

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

type JSON struct {
	Port string `json:"P"`
	Data []Data
}
type Data struct {
	Data string `json:"D"`
	ID   string `json:"Id"`
}

func (sp *SPJS) SendJSON(v JSON) {
	data, err := json.Marshal(v)
	if err != nil {
		// shouldn't happen since we control everything that's sent out
		log.Panicln("ERROR: sendjson (marshal):", err)
		return
	}

	ch := make(chan struct{})
	sp.outgoing <- message{done: ch, payload: append([]byte("sendjson "), data...)}
	<-ch
}
func (sp *SPJS) WriteString(data string) {
	ch := make(chan struct{})
	sp.outgoing <- message{done: ch, payload: []byte(data)}
	<-ch
}
