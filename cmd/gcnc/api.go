package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	sse "github.com/alexandrevicenzi/go-sse"
)

type api struct {
	http.Handler
	m       Machine
	dataDir string
	sse     *sse.Server
}

func newAPI(m Machine, dir string) *api {
	mux := http.NewServeMux()

	a := &api{
		Handler: mux,
		m:       m,
		dataDir: dir,
		sse: sse.NewServer(&sse.Options{
			Logger: log.New(ioutil.Discard, "", 0),
		}),
	}

	fs := http.FileServer(http.Dir(dir))
	mux.Handle("/data/", http.StripPrefix("/data", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "GET":
			fs.ServeHTTP(w, req)
		case "PUT":
			a.putFile(w, req)
		case "DELETE":
			a.deleteFile(w, req)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	})))

	mux.HandleFunc("/api/run", a.run)
	mux.HandleFunc("/api/probe", a.probe)

	mux.Handle("/events/", a.sse)
	go func() {
		for state := range m.State() {
			data, err := json.Marshal(state)
			if err != nil {
				log.Printf("ERROR: marshal json: %+v", err)
				continue
			}
			a.sse.SendMessage("/events/state", sse.SimpleMessage(string(data)))
		}
	}()

	return a
}

func safePath(base, name string) (bool, string) {
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		log.Println("invalid path '" + name + "'")
		return false, ""
	}
	dir := string(base)
	if dir == "" {
		dir = "."
	}
	fullName := filepath.Join(dir, filepath.FromSlash(path.Clean("/"+name)))
	return true, fullName
}

func (a *api) run(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return
	}

	parts := strings.Split(string(data), "\n")
	p := parts[:0]
	for _, str := range parts {
		str = strings.TrimSpace(str)
		if str == "" {
			continue
		}
		p = append(p, str+"\n")
	}
	err = a.m.Run(p)
	if err != nil {
		log.Printf("ERROR: run: %+v", err)
		http.Error(w, err.Error(), 500)
		return
	}
}

func (a *api) probe(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	ok, name := safePath(a.dataDir, "grid.json")
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var err error
	var opt ProbeOptions
	opt.ZeroZAxis = req.FormValue("zeroZAxis") == "1"

	parse := func(param string) (val float64) {
		if err != nil {
			return 0
		}
		val, err = strconv.ParseFloat(req.FormValue(param), 64)
		return val
	}
	opt.FeedRate = parse("feedRate")
	opt.MaxTravel = parse("maxZTravel")

	grid := req.FormValue("grid") == "1"
	var xDist, yDist float64
	if grid {
		xDist = parse("xDist")
		yDist = parse("yDist")
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var res interface{}
	if grid {
		res, err = a.m.ProbeGrid(opt, xDist, yDist)
	} else {
		res, err = a.m.ProbeZ(opt)
	}

	if err != nil {
		log.Printf("ERROR: probe grid=%t: %+v", grid, err)
		http.Error(w, err.Error(), 500)
		return
	}

	out := io.Writer(w)
	if grid {
		os.MkdirAll(filepath.Dir(name), 0755)
		f, err := os.Create(name)
		if err != nil {
			log.Printf("ERROR: create '%s': %+v", name, err)
		} else {
			defer f.Close()
			out = io.MultiWriter(w, f)
		}
	}
	err = json.NewEncoder(out).Encode(res)
	if err != nil {
		log.Println("ERROR: encode:", err)
	}
}

func (a *api) putFile(w http.ResponseWriter, req *http.Request) {
	ok, name := safePath(a.dataDir, req.URL.Path)
	if !ok {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	os.MkdirAll(filepath.Dir(name), 0755)
	f, err := os.Create(name)
	if err != nil {
		log.Printf("ERROR: create '%s': %+v", name, err)
		http.Error(w, err.Error(), 500)
		return
	}
	defer f.Close()
	_, err = io.Copy(f, req.Body)
	if err != nil {
		log.Printf("ERROR: write '%s': %+v", name, err)
		http.Error(w, err.Error(), 500)
		return
	}
}
func (a *api) deleteFile(w http.ResponseWriter, req *http.Request) {
	ok, name := safePath(a.dataDir, req.URL.Path)
	if !ok {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	err := os.Remove(name)
	if err != nil {
		log.Printf("ERROR: delete '%s': %+v", name, err)
		http.Error(w, err.Error(), 500)
		return
	}
}
