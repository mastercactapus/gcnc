package grbl

import (
	"errors"
	"strconv"
	"strings"

	"github.com/mastercactapus/gcnc/coord"
	"github.com/mastercactapus/gcnc/machine"
)

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

func parseProbe(data string) (*machine.ProbeResult, error) {
	data = strings.TrimSpace(data)
	data = strings.TrimPrefix(data, "[")
	data = strings.TrimSuffix(data, "]")
	parts := strings.Split(data, ":")
	var err error
	switch parts[0] {
	case "PRB":
		var res machine.ProbeResult
		res.Valid = parts[2] == "1"
		res.Point, err = parseCoords(parts[1])
		if err != nil {
			return nil, err
		}

		return &res, nil
	}

	return nil, errors.New("unknown PUSH message: " + data)
}

func parseStatus(stat machine.State, data string) (*machine.State, error) {
	data = strings.TrimSpace(data)
	data = strings.TrimPrefix(data, "<")
	data = strings.TrimSuffix(data, ">")
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
			return nil, err
		}
	}
	return &stat, nil
}
