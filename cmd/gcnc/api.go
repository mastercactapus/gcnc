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
	"github.com/mastercactapus/gcnc/coord"
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
		case "OPTIONS":
			return
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

	mux.HandleFunc("/api/tool/change", a.toolChange)

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

	go func() {
		for msg := range m.HoldMessage() {
			a.sse.SendMessage("/events/hold", sse.SimpleMessage(msg))
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

	grid := req.URL.Query().Get("gridLevel")
	if grid != "" {
		lvl, err := strconv.ParseFloat(grid, 64)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		ok, gridFile := safePath(a.dataDir, "grid.json")
		if !ok {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		data, err := ioutil.ReadFile(gridFile)
		if err != nil {
			log.Println("ERROR: read grid.json:", err)
			http.Error(w, err.Error(), 400)
			return
		}
		var gridData []coord.Point
		err = json.Unmarshal(data, &gridData)
		if err != nil {
			log.Println("ERROR: parse grid.json:", err)
			http.Error(w, err.Error(), 500)
			return
		}
		err = a.m.RunLevel(p, lvl, gridData)
	} else {
		err = a.m.Run(p)
	}

	if err != nil {
		log.Printf("ERROR: run: gridLevel=%s %+v", grid, err)
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

	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return
	}

	var res interface{}
	grid := req.URL.Query().Get("grid") == "1"
	if grid {
		var opt GridOptions
		err = json.Unmarshal(data, &opt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		res, err = a.m.ProbeGrid(opt)
	} else {
		var opt ProbeOptions
		err = json.Unmarshal(data, &opt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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