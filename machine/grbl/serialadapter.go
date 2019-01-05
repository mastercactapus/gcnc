package grbl

import (
	"io"
	"log"
	"sync"
	"time"

	"github.com/mastercactapus/gcnc/machine"
)

type SerialAdapter struct {
	*Conn

	mx      sync.Mutex
	last    machine.State
	state   chan machine.State
	message chan string
	data    chan string

	probes      []machine.ProbeResult
	getProbes   chan []machine.ProbeResult
	resetProbes chan struct{}
}

var _ machine.Adapter = &SerialAdapter{}

func NewSerialAdapter(rw io.ReadWriter) *SerialAdapter {
	conn := NewConn(rw)
	go func() {
		for range time.NewTicker(500 * time.Millisecond).C {
			conn.WriteByte('?')
		}
	}()
	adapter := &SerialAdapter{
		Conn: conn,

		state:       make(chan machine.State),
		getProbes:   make(chan []machine.ProbeResult),
		resetProbes: make(chan struct{}),
		message:     make(chan string),
		data:        make(chan string),
	}
	go adapter.loop()
	go adapter.readLoop()

	return adapter
}
func (adapter *SerialAdapter) Probes() []machine.ProbeResult { return <-adapter.getProbes }

func (adapter *SerialAdapter) ResetProbes() { adapter.resetProbes <- struct{}{} }

func (adapter *SerialAdapter) readLoop() {
	buf := make([]byte, 1024)
	for {
		n, err := adapter.Read(buf)
		if err != nil {
			log.Println("ERROR: read from port: ", err)
			continue
		}
		adapter.data <- string(buf[:n])
	}
}
func (adapter *SerialAdapter) State() chan machine.State { return adapter.state }
func (adapter *SerialAdapter) CurrentState() machine.State {
	adapter.mx.Lock()
	state := adapter.last
	adapter.mx.Unlock()
	return state
}
func (adapter *SerialAdapter) loop() {
	for {
		select {
		case <-adapter.resetProbes:
			adapter.probes = nil
		case adapter.getProbes <- adapter.probes:
		case data := <-adapter.data:
			if len(data) == 0 {
				continue
			}
			if data[0] == '<' {
				stat, err := parseStatus(adapter.last, data)
				if err != nil {
					log.Println("ERROR: parse status:", err)
					continue
				}
				adapter.mx.Lock()
				adapter.last = *stat
				adapter.mx.Unlock()
				select {
				case adapter.state <- adapter.last:
				default:
				}
			} else if data[0] == '[' {
				prb, err := parseProbe(data)
				if err != nil {
					log.Println("ERROR: parse:", err)
					continue
				}
				adapter.probes = append(adapter.probes, *prb)
			}
		}
	}
}
