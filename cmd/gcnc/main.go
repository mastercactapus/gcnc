package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/mastercactapus/gcnc/machine"
	"github.com/mastercactapus/gcnc/machine/grbl"
	"github.com/mastercactapus/gcnc/spjs"
)

func main() {
	log.SetFlags(log.Lshortfile)

	port := flag.String("port", "/dev/ttyUSB0", "Port path (or name if using SPJS).")
	spjsURL := flag.String("spjs", "ws://cnc-bridge:8989/ws", "Websocket URL of the SPJS server to use.")
	controller := flag.String("controller", "grbl", "Name of the controller to use.")
	addr := flag.String("addr", ":9091", "Address to bind the gCNC server to.")
	dir := flag.String("dir", "./data", "Data directory to use.")
	flag.Parse()

	if *controller != "grbl" {

	}

	var sp *spjs.SPJS
	if *spjsURL != "" {
		sp := spjs.NewSPJS(*spjsURL)
	}

	var adapter machine.Adapter
	switch *controller {
	case "grbl":
		if sp != nil {
			adapter = grbl.NewSPJSAdapter(sp, *port)
		} else {
			adapter = grbl.NewSerialAdapter(nil)
		}
	default:
		log.Fatal("only 'grbl' controller supported")
	}

	m := machine.NewMachine(adapter)

	api := newAPI(m, *dir)

	err := http.ListenAndServe(*addr, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		log.Printf("%s %s - %s", req.Method, req.URL.Path, req.RemoteAddr)
		api.ServeHTTP(w, req)
	}))
	if err != nil {
		log.Fatal(err)
	}
}
