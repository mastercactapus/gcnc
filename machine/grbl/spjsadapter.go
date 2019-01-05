package grbl

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/mastercactapus/gcnc/machine"
	"github.com/mastercactapus/gcnc/spjs"
)

var lastID int64

func nextID() string {
	id := atomic.AddInt64(&lastID, 1)
	return "cmd_" + strconv.FormatInt(id, 36)
}

type SPJSAdapter struct {
	sp   *spjs.SPJS
	port string

	cmds    chan adapterMessage
	waiting map[string]chan error

	mx      sync.Mutex
	last    machine.State
	state   chan machine.State
	message chan string

	probes      []machine.ProbeResult
	getProbes   chan []machine.ProbeResult
	resetProbes chan struct{}
}

var _ machine.Adapter = &SPJSAdapter{}

type adapterMessage struct {
	spjs.JSON
	wait chan error
}

func NewSPJSAdapter(sp *spjs.SPJS, port string) *SPJSAdapter {
	adapter := &SPJSAdapter{
		sp:        sp,
		port:      port,
		waiting:   make(map[string]chan error, 100),
		cmds:      make(chan adapterMessage, 1000),
		state:     make(chan machine.State),
		getProbes: make(chan []machine.ProbeResult),
		message:   make(chan string),
	}
	go adapter.loop()

	return adapter
}
func (adapter *SPJSAdapter) Probes() []machine.ProbeResult { return <-adapter.getProbes }

func (adapter *SPJSAdapter) ResetProbes() { adapter.resetProbes <- struct{}{} }

func (adapter *SPJSAdapter) CurrentState() machine.State {
	adapter.mx.Lock()
	defer adapter.mx.Unlock()
	return adapter.last
}
func (adapter *SPJSAdapter) setMachineState(state machine.State) {
	adapter.mx.Lock()
	defer adapter.mx.Unlock()
	adapter.last = state
	select {
	case adapter.state <- state:
	default:
	}
}
func (adapter *SPJSAdapter) loop() {
	for {
		select {
		case adapter.getProbes <- adapter.probes:
		case <-adapter.resetProbes:
			adapter.probes = nil
		case resp := <-adapter.sp.Messages():
			switch msg := resp.(type) {
			case *spjs.DataFrame:
				if msg.Data[0] == '<' {
					stat, err := parseStatus(adapter.last, msg.Data)
					if err != nil {
						log.Println("ERROR: parse status:", err)
						continue
					}
					adapter.setMachineState(*stat)
				} else if msg.Data[0] == '[' {
					prb, err := parseProbe(msg.Data)
					if err != nil {
						log.Println("ERROR: parse:", err)
						continue
					}
					adapter.probes = append(adapter.probes, *prb)
				}
			case *spjs.CmdStatus:
				switch msg.Cmd {
				case "WipedQueue":
					for key, ch := range adapter.waiting {
						ch <- errors.New("wiped queue")
						delete(adapter.waiting, key)
					}
				case "Complete":
					if adapter.waiting[msg.ID] != nil {
						adapter.waiting[msg.ID] <- nil
						delete(adapter.waiting, msg.ID)
					}
				}
			case *spjs.SerialPortList:
				for _, port := range msg.SerialPorts {
					if port.Name != adapter.port {
						continue
					}
					if !port.IsOpen {
						adapter.sp.WriteString("open " + adapter.port + " grbl 115200")
					}
				}
			}
		case msg := <-adapter.cmds:
			adapter.sp.SendJSON(msg.JSON)
			if msg.wait != nil {
				adapter.waiting[msg.Data[len(msg.Data)-1].ID] = msg.wait
			}
		}
	}
}

func (adapter *SPJSAdapter) State() chan machine.State {
	return adapter.state
}

func (adapter *SPJSAdapter) ReadFrom(r io.Reader) (n int64, err error) {
	scan := bufio.NewScanner(r)
	var wait chan error
	for {
		var j spjs.JSON
		j.Port = adapter.port
		for scan.Scan() {
			n += int64(len(scan.Bytes()))
			j.Data = append(j.Data, spjs.Data{
				Data: strings.TrimSpace(scan.Text()) + "\n",
				ID:   nextID(),
			})
			if len(j.Data) == 100 {
				break
			}
		}
		if len(j.Data) == 0 {
			break
		}
		wait = make(chan error, 1)
		adapter.cmds <- adapterMessage{JSON: j, wait: wait}
	}

	if wait == nil {
		return 0, nil
	}

	// wait for last channel
	return n, <-wait
}
func (adapter *SPJSAdapter) WriteByte(b byte) error {
	_, err := adapter.Write([]byte(string(b) + "\n"))
	return err
}
func (adapter *SPJSAdapter) Write(p []byte) (int, error) {
	n, err := adapter.ReadFrom(bytes.NewBuffer(p))
	return int(n), err
}
