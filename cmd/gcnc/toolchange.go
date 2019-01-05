package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/mastercactapus/gcnc/machine"
)

func (a *api) toolChange(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return
	}
	var opt machine.ToolChangeOptions
	err = json.Unmarshal(data, &opt)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	err = a.m.ToolChange(opt)
	if err != nil {
		log.Printf("ERROR: tool-change: %+v", err)
		http.Error(w, err.Error(), 500)
		return
	}
}
