package main

import (
	"bytes"
	"errors"
	"io"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/mastercactapus/gcnc/coord"
	"github.com/mastercactapus/gcnc/gcode"
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
	waiting map[string]chan error

	mx      sync.Mutex
	last    MachineState
	state   chan MachineState
	message chan string

	probes    []ProbeResult
	getProbes chan []ProbeResult
}
type adapterMessage struct {
	spjsJSON
	wait chan error
}

func newGrblAdapter(sp *spjs, port string) *spjsGrblAdapter {
	adapter := &spjsGrblAdapter{
		sp:        sp,
		port:      port,
		waiting:   make(map[string]chan error, 100),
		cmds:      make(chan adapterMessage, 1000),
		state:     make(chan MachineState),
		getProbes: make(chan []ProbeResult),
		message:   make(chan string),
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

func (adapter *spjsGrblAdapter) MachineState() MachineState {
	adapter.mx.Lock()
	defer adapter.mx.Unlock()
	return adapter.last
}
func (adapter *spjsGrblAdapter) SetMachineState(state MachineState) {
	adapter.mx.Lock()
	defer adapter.mx.Unlock()
	adapter.last = state
	adapter.state <- state
}
func (adapter *spjsGrblAdapter) setMessage(data string) {
	adapter.message <- data
}
func (adapter *spjsGrblAdapter) parseStatus(data string) {
	data = strings.TrimSpace(data)
	data = strings.TrimPrefix(data, "<")
	data = strings.TrimSuffix(data, ">")
	stat := adapter.MachineState()
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
	adapter.SetMachineState(stat)
}
func (adapter *spjsGrblAdapter) parsePush(data string) {
	data = strings.TrimSpace(data)
	data = strings.TrimPrefix(data, "[")
	data = strings.TrimSuffix(data, "]")
	parts := strings.Split(data, ":")
	var err error
	switch parts[0] {
	case "PRB":
		var res ProbeResult
		res.Valid = parts[2] == "1"
		res.Point, err = parseCoords(parts[1])
		if err != nil {
			log.Println("ERROR: parse PRB message:", err)
		}
		adapter.probes = append(adapter.probes, res)
		return
	}

	log.Println("ERROR: unknown PUSH message:", data)
}
func (adapter *spjsGrblAdapter) RunLevel(data []string, granularity float64, points []coord.Point) error {
	stat := adapter.MachineState()
	if stat.Status != "Idle" {
		return nil, errors.New("machine not idle")
	}

	mesh, err := meshlevel.NewMesh(points)
	if err != nil {
		return error
	}
	cfg := meshlevel.Config{
		ZOffsetter: mesh,

		MPos: stat.MPos,
		WCO:  stat.WCO,

		Granularity: granularity,
		Reader: gcode.NewParser(
			bytes.NewBufferString(strings.Join(data, "\n")),
		),
	}

	r := gcode.NewBuffer(meshlevel.New(cfg))
	var buf bytes.Buffer

	_, err = io.Copy(&buf, r)
	if err != nil {
		return err
	}

	return adapter.Run(strings.Split(buf.String(), "\n"))
}
func (adapter *spjsGrblAdapter) loop() {
	for {
		select {
		case adapter.getProbes <- adapter.probes:
			adapter.probes = nil
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

func (adapter *spjsGrblAdapter) ProbeGrid(opt GridOptions) ([]ProbeResult, error) {
	stat := adapter.MachineState()
	if stat.Status != "Idle" {
		return nil, errors.New("machine not idle")
	}

	<-adapter.getProbes
	err := adapter.runBlocks(opt.GenerateGridQuick(stat.MPos))
	if err != nil {
		return nil, err
	}

	startProbes := <-adapter.getProbes
	if len(startProbes) == 0 {
		return nil, errors.New("no probe data returned")
	}

	maxZ := startProbes[0].Z
	for _, p := range startProbes[1:] {
		maxZ = math.Max(maxZ, p.Z)
	}
	maxZ += 0.2

	err = adapter.runBlocks(opt.GenerateGridSequence(stat.MPos, maxZ))
	if err != nil {
		return nil, err
	}

	granProbes := <-adapter.getProbes

	return append(startProbes, granProbes...), nil
}
func (adapter *spjsGrblAdapter) toolProbe(opt ToolChangeOptions) (*ProbeResult, error) {
	err := adapter.runBlocks(generateGoTo(opt.TravelHeight, opt.ProbePos))
	if err != nil {
		return nil, err
	}

	p, err := adapter.ProbeZ(ProbeOptions{Wait: true, MaxTravel: opt.MaxTravel, FeedRate: opt.FeedRate})
	if err != nil {
		return nil, err
	}
	if !p.Valid {
		return nil, errors.New("tool probe failed")
	}
	return p, nil
}
func (adapter *spjsGrblAdapter) HoldMessage() chan string {
	return adapter.message
}
func (adapter *spjsGrblAdapter) Hold(message string) error {
	adapter.setMessage(message)
	err := adapter.Run([]string{"M0"})
	adapter.setMessage("-")
	if err != nil {
		return err
	}
	return nil
}
func (adapter *spjsGrblAdapter) ToolChange(opt ToolChangeOptions) error {
	stat := adapter.MachineState()
	if stat.Status != "Idle" {
		return errors.New("machine not idle")
	}

	if opt.LastToolPos == nil {
		// get current tool first
		p, err := adapter.toolProbe(opt)
		if err != nil {
			return err
		}
		opt.LastToolPos = &p.Point
		err = adapter.Hold("Probe complete, remove Z-Probe.")
		if err != nil {
			return err
		}
	}

	err := adapter.runBlocks(generateGoTo(opt.TravelHeight, opt.ChangePos))
	if err != nil {
		return err
	}

	adapter.Hold("Perform tool change.")
	if err != nil {
		return err
	}

	p, err := adapter.toolProbe(opt)
	if err != nil {
		return err
	}

	diff := opt.LastToolPos.Z - p.Z
	stat.MPos.Z -= diff
	WPos := stat.MPos.Sub(stat.WCO)
	WPos.Z -= diff
	log.Println("Adjusting Z-offset by:", diff)

	err = adapter.runBlocks([]gcode.Block{
		{
			{W: 'G', Arg: 92},
			{W: 'Z', Arg: WPos.Z},
		},
	})
	if err != nil {
		return err
	}

	adapter.Hold("Probe complete, remove Z-Probe.")
	if err != nil {
		return err
	}

	err = adapter.runBlocks(generateGoTo(opt.TravelHeight, stat.MPos))
	if err != nil {
		return err
	}

	return nil
}

func (adapter *spjsGrblAdapter) ProbeZ(opt ProbeOptions) (*ProbeResult, error) {
	if opt.Wait {
		err := adapter.Hold("Attach Z-Probe to spindle.")
		if err != nil {
			return nil, err
		}
	}
	stat := adapter.MachineState()
	if stat.Status != "Idle" && stat.Status != "Hold:0" {
		return nil, errors.New("machine not idle")
	}

	<-adapter.getProbes
	err := adapter.runBlocks(opt.Generate(stat.MPos))
	if err != nil {
		return nil, err
	}
	p := <-adapter.getProbes
	if len(p) == 0 {
		return nil, errors.New("no probe data returned")
	}

	return &p[0], nil
}

func (adapter *spjsGrblAdapter) State() chan MachineState {
	return adapter.state
}

func (adapter *spjsGrblAdapter) runBlocks(b []gcode.Block) error {
	str := make([]string, len(b))
	for i, blk := range b {
		str[i] = blk.String()
	}
	return adapter.Run(str)
}
func (adapter *spjsGrblAdapter) Run(data []string) error {
	if len(data) == 0 {
		return nil
	}

	var j spjsJSON
	j.Port = adapter.port
	for _, dat := range data {
		j.Data = append(j.Data, spjsData{
			Data: strings.TrimSpace(dat) + "\n",
			ID:   nextID(),
		})
	}

	ch := make(chan error, 1)
	adapter.cmds <- adapterMessage{spjsJSON: j, wait: ch}
	return <-ch
}
