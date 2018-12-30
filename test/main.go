package main

import (
	"strings"

	"github.com/joushou/gocnc/gcode"
	"github.com/joushou/gocnc/vm"
)

func main() {
	doc, err := gcode.Parse(strings.TrimSpace(`
		G91 X10
	
	`))
	if err != nil {
		panic(err)
	}

	var m vm.Machine
	m.Init()
	m.Process(doc)
	m.Dump()
}
