package main

import (
	"errors"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/mastercactapus/gcnc/coord"
)

var lastID int64

func nextID() string {
	id := atomic.AddInt64(&lastID, 1)
	return "cmd_" + strconv.FormatInt(id, 36)
}

type spjsGrblAdapter struct {
	sp   *spjs
	port string

	cmds    chan adapterMessage
	mx      sync.Mutex
	waiting map[string]chan error

	last  MachineState
	state chan MachineState
}
type adapterMessage struct {
	spjsJSON
	wait chan error
}

func newGrblAdapter(sp *spjs, port string) *spjsGrblAdapter {
	adapter := &spjsGrblAdapter{
		sp:      sp,
		port:    port,
		waiting: make(map[string]chan error, 100),
		cmds:    make(chan adapterMessage, 1000),
		state:   make(chan MachineState),
	}
	go adapter.loop()

	return adapter
}
func parseCoords(data string) (p coord.Point, err error) {
	parts := strings.Split(data, ",")
	if len(parts) != 3 {
		return p, errors.New("invalid number of elements")
	}
	p.X, err = strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return p, err
	}
	p.Y, err = strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return p, err
	}
	p.Z, err = strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return p, err
	}
	return p, nil
}
func (adapter *spjsGrblAdapter) parseStatus(data string) {
	data = strings.TrimSpace(data)
	data = strings.TrimPrefix(data, "<")
	data = strings.TrimSuffix(data, ">")
	stat := adapter.last
	parts := strings.Split(data, "|")
	stat.Status = parts[0]
	var err error
	for _, s := range parts[1:] {
		sParts := strings.SplitN(s, ":", 2)
		switch sParts[0] {
		case "MPos":
			stat.MPos, err = parseCoords(sParts[1])
		case "WCO":
			stat.WCO, err = parseCoords(sParts[1])
		}
		if err != nil {
			log.Println("ERROR: parse status:", err)
			return
		}
	}
	adapter.last = stat
	adapter.state <- stat
}
func (adapter *spjsGrblAdapter) parsePush(data string) {

}
func (adapter *spjsGrblAdapter) loop() {
	for {
		select {
		case resp := <-adapter.sp.incomming:
			switch msg := resp.(type) {
			case *spjsDataFrame:
				if msg.Data[0] == '<' {
					adapter.parseStatus(msg.Data)
				} else if msg.Data[0] == '[' {
					adapter.parsePush(msg.Data)
				}
			case *spjsCmdQStatus:
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
			case *spsjsPorts:
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
			adapter.sp.SendJSON(msg.spjsJSON)
			adapter.waiting[msg.Data[len(msg.Data)-1].ID] = msg.wait
		}
	}
}

func (adapter *spjsGrblAdapter) ProbeGrid(opt ProbeOptions, x, y float64) ([]ProbeResult, error) {
	return nil, errors.New("not implemented")
}

func (adapter *spjsGrblAdapter) ProbeZ(opt ProbeOptions) (*ProbeResult, error) {
	return nil, errors.New("not implemented")
}
func (adapter *spjsGrblAdapter) State() chan MachineState {
	return adapter.state
}

func (adapter *spjsGrblAdapter) Run(data []string) error {
	if len(data) == 0 {
		return nil
	}
	var j spjsJSON
	j.Port = adapter.port
	for _, dat := range data {
		j.Data = append(j.Data, spjsData{
			Data: dat,
			ID:   nextID(),
		})
	}

	ch := make(chan error, 1)
	adapter.cmds <- adapterMessage{spjsJSON: j, wait: ch}
	return <-ch
}
