package gcode

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

type Parser struct{ br *bufio.Reader }

func NewParser(r io.Reader) *Parser {
	if br, ok := r.(*bufio.Reader); ok {
		return &Parser{br: br}
	}

	return &Parser{br: bufio.NewReader(r)}
}

var (
	rx      = regexp.MustCompile(`^([A-Z][0-9.\-]+)+$`)
	rxSplit = regexp.MustCompile(`[A-Z][0-9.\-]+`)
)

func (p *Parser) Read() (ln Block, err error) {
	for {
		s, err := p.br.ReadString('\n')
		if err == io.EOF && s != "" {
			err = nil
		}
		if err != nil {
			return nil, err
		}

		s = strings.SplitN(s, ";", 2)[0]
		s = strings.Replace(s, " ", "", -1)
		s = strings.TrimSpace(s)
		s = strings.ToUpper(s)

		if s == "" {
			continue
		}

		if !rx.MatchString(s) {
			return nil, errors.New("invalid or unhandled line: " + s)
		}

		codes := rxSplit.FindAllString(s, -1)
		res := make([]Word, len(codes))

		for i, c := range codes {
			_, err = fmt.Sscanf(c, "%c%f", &res[i].W, &res[i].Arg)
			if err != nil {
				return nil, err
			}
		}

		return res, nil
	}
}
