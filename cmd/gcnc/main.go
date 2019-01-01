package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
)

func main() {
	log.SetFlags(log.Lshortfile)

	port := flag.String("port", "spjs://cnc-bridge:8989", "Path or URL to the CNC.")
	addr := flag.String("addr", ":9091", "Address to bind the gCNC server to.")
	dir := flag.String("dir", "./data", "Data directory to use.")
	flag.Parse()

	u, err := url.Parse(*port)
	if err != nil {
		log.Fatalln("invalid port url:", err)
	}
	switch u.Scheme {
	case "spjs":
		u.Scheme = "ws"
		u.Path = "/ws"
	}

	sp := newSPJS(u.String())

	adapter := newGrblAdapter(sp, "/dev/ttyUSB0")

	api := newAPI(adapter, *dir)

	err = http.ListenAndServe(*addr, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		log.Printf("%s %s - %s", req.Method, req.URL.Path, req.RemoteAddr)
		api.ServeHTTP(w, req)
	}))
	if err != nil {
		log.Fatal(err)
	}
}
