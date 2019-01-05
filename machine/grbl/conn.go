package grbl

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
)

const bufferSize = 128

// ErrGrblReset will be returned from write methods if a reset is encountered
// before all commands are run.
var ErrGrblReset = errors.New("grbl reset")

// Conn represents a direct connection to a Grbl controller.
type Conn struct {
	rw io.ReadWriter

	readBuf []byte
	scan    *bufio.Scanner
	ackCh   chan error
	resetCh chan struct{}
	closeCh chan struct{}

	mx  sync.Mutex
	wMx sync.Mutex

	deviceBuf int
	lineSize  []int

	wroteLines int64
	readLines  int64
}

// NewConn creates a new Conn using the provided ReadWriter for data.
func NewConn(rw io.ReadWriter) *Conn {
	return &Conn{
		scan:    bufio.NewScanner(rw),
		rw:      rw,
		ackCh:   make(chan error),
		resetCh: make(chan struct{}, 1),
	}
}

// Close will abort any in-progress writes and close the
// underlying ReadWriter, if it implements io.Closer.
func (c *Conn) Close() error {
	close(c.closeCh)
	if closer, ok := c.rw.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (c *Conn) recordBufferSpace(n int) int64 {
	c.deviceBuf += n
	c.wroteLines++
	c.lineSize = append(c.lineSize, n)
	return c.wroteLines
}

func (c *Conn) waitForBufferSpace(n int) error {
	for c.deviceBuf+n > bufferSize {
		err := c.next()
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Conn) next() error {
	select {
	case <-c.closeCh:
		return io.ErrClosedPipe
	default:
	}

	select {
	case <-c.resetCh:
		c.deviceBuf = 0
		c.lineSize = nil
		c.readLines = c.wroteLines
		return ErrGrblReset
	default:
	}

	select {
	case <-c.closeCh:
		return io.ErrClosedPipe
	case <-c.resetCh:
		c.deviceBuf = 0
		c.lineSize = nil
		c.readLines = c.wroteLines
		return ErrGrblReset
	case e := <-c.ackCh:
		c.readLines++
		c.deviceBuf -= c.lineSize[0]
		c.lineSize = c.lineSize[1:]
		return e
	}
}

func (c *Conn) waitForLine(id int64) (err error) {
	for {
		e := c.next()
		if err == nil {
			err = e
		}
		if c.readLines == id {
			return err
		}
	}
}

// writeLine will block until line has been written to the serial device in full.
//
// It returns the line index.
func (c *Conn) writeLine(line []byte) (id int64, err error) {
	err = c.waitForBufferSpace(len(line))
	if err != nil {
		return 0, err
	}
	c.mx.Lock()
	_, err = c.rw.Write(line)
	c.mx.Unlock()
	if err != nil {
		return 0, err
	}
	id = c.recordBufferSpace(len(line))
	return id, nil
}

func splitLinesKeepN(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, data[:i+1], nil
	}
	if atEOF {
		return len(data), data, io.ErrUnexpectedEOF
	}
	return 0, nil, nil
}

// ReadFrom returns after all lines have been sent and executed.
func (c *Conn) ReadFrom(r io.Reader) (n int64, err error) {
	c.wMx.Lock()
	defer c.wMx.Unlock()
	select {
	case <-c.closeCh:
		return 0, io.ErrClosedPipe
	default:
	}

	scanner := bufio.NewScanner(r)
	scanner.Split(splitLinesKeepN)

	lastID := c.wroteLines
	for scanner.Scan() {
		lastID, err = c.writeLine(scanner.Bytes())
		scanner.Bytes()
		if err != nil {
			return n, err
		}
		n += int64(len(scanner.Bytes()))
	}

	return n, c.waitForLine(lastID)
}

// Write will return after all lines have been sent and executed.
func (c *Conn) Write(p []byte) (int, error) {
	c.wMx.Lock()
	defer c.wMx.Unlock()

	n, err := c.ReadFrom(bytes.NewBuffer(p))
	return int(n), err
}

// WriteByte will write directly to the serial device without
// accounting for buffering.
//
// Use for realtime commands like `?`.
func (c *Conn) WriteByte(p byte) (err error) {
	select {
	case <-c.closeCh:
		return io.ErrClosedPipe
	default:
	}
	c.mx.Lock()
	_, err = c.rw.Write([]byte{p})
	c.mx.Unlock()
	return err
}

// Read will read the next line from the device.
func (c *Conn) Read(p []byte) (n int, err error) {
	select {
	case <-c.closeCh:
		return 0, io.ErrClosedPipe
	default:
	}

	if c.readBuf != nil {
		if len(p) < len(c.readBuf) {
			return 0, io.ErrShortBuffer
		}
		n = copy(p, c.readBuf)
		c.readBuf = nil
		return n, nil
	}
	if !c.scan.Scan() {
		return 0, c.scan.Err()
	}
	data := c.scan.Bytes()

	if bytes.Equal(data, []byte("ok")) {
		select {
		case c.ackCh <- nil:
		case <-c.closeCh:
			return n, io.ErrClosedPipe
		}
	} else if bytes.HasPrefix(data, []byte("error:")) {
		select {
		case c.ackCh <- errors.New(strings.TrimSpace(string(data))):
		case <-c.closeCh:
			return n, io.ErrClosedPipe
		}
	} else if bytes.HasPrefix(data, []byte("Grbl")) {
		select {
		case c.resetCh <- struct{}{}:
		default:
		}
	}

	if len(p) < len(data) {
		c.readBuf = data
		return 0, io.ErrShortBuffer
	}

	return copy(p, data), nil
}
