// Package udp provides a connection-oriented listener over a UDP PacketConn
package udp

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pion/ion/pkg/rtc/rtpengine/deadline"
)

const receiveMTU = 8192

var errClosedListener = errors.New("udp: listener closed")

// Listener augments a connection-oriented Listener over a UDP PacketConn
type Listener struct {
	pConn *net.UDPConn

	accepting atomic.Value // bool
	acceptCh  chan *Conn
	doneCh    chan struct{}
	doneOnce  sync.Once

	connLock sync.Mutex
	conns    map[string]*Conn
	connWG   sync.WaitGroup

	readWG   sync.WaitGroup
	errClose atomic.Value // error
}

// Accept waits for and returns the next connection to the listener.
func (l *Listener) Accept() (*Conn, error) {
	select {
	case c := <-l.acceptCh:
		l.connWG.Add(1)
		return c, nil

	case <-l.doneCh:
		return nil, errClosedListener
	}
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *Listener) Close() error {
	var err error
	l.doneOnce.Do(func() {
		l.connWG.Done()
		l.accepting.Store(false)
		close(l.doneCh)

		l.connLock.Lock()
		nConns := len(l.conns)
		l.connLock.Unlock()

		if nConns == 0 {
			// Wait if this is the final connection
			l.readWG.Wait()
			if errClose, ok := l.errClose.Load().(error); ok {
				err = errClose
			}
		} else {
			err = nil
		}
	})

	return err
}

// Addr returns the listener's network address.
func (l *Listener) Addr() net.Addr {
	return l.pConn.LocalAddr()
}

// Listen creates a new listener
func Listen(network string, laddr *net.UDPAddr) (*Listener, error) {
	conn, err := net.ListenUDP(network, laddr)
	if err != nil {
		return nil, err
	}

	l := &Listener{
		pConn:    conn,
		acceptCh: make(chan *Conn),
		conns:    make(map[string]*Conn),
		doneCh:   make(chan struct{}),
	}
	l.accepting.Store(true)
	l.connWG.Add(1)
	l.readWG.Add(2) // wait readLoop and Close execution routine

	go l.readLoop()
	go func() {
		l.connWG.Wait()
		if err := l.pConn.Close(); err != nil {
			l.errClose.Store(err)
		}
		l.readWG.Done()
	}()

	return l, nil
}

// readLoop has to tasks:
// 1. Dispatching incoming packets to the correct Conn.
//    It can therefore not be ended until all Conns are closed.
// 2. Creating a new Conn when receiving from a new remote.
func (l *Listener) readLoop() {
	defer l.readWG.Done()
	buf := make([]byte, receiveMTU)

	for {
		n, raddr, err := l.pConn.ReadFrom(buf)
		if err != nil {
			return
		}
		conn, err := l.getConn(raddr)
		if err != nil {
			continue
		}
		cBuf := <-conn.readCh
		n = copy(cBuf, buf[:n])
		conn.sizeCh <- n
	}
}

func (l *Listener) getConn(raddr net.Addr) (*Conn, error) {
	l.connLock.Lock()
	defer l.connLock.Unlock()
	conn, ok := l.conns[raddr.String()]
	if !ok {
		if !l.accepting.Load().(bool) {
			return nil, errClosedListener
		}
		conn = l.newConn(raddr)
		l.conns[raddr.String()] = conn
		l.acceptCh <- conn
	}
	return conn, nil
}

// Conn augments a connection-oriented connection over a UDP PacketConn
type Conn struct {
	listener *Listener

	rAddr net.Addr

	readCh chan []byte
	sizeCh chan int

	doneCh   chan struct{}
	doneOnce sync.Once

	readDeadline  *deadline.Deadline
	writeDeadline *deadline.Deadline
}

func (l *Listener) newConn(rAddr net.Addr) *Conn {
	return &Conn{
		listener:      l,
		rAddr:         rAddr,
		readCh:        make(chan []byte),
		sizeCh:        make(chan int),
		doneCh:        make(chan struct{}),
		readDeadline:  deadline.New(),
		writeDeadline: deadline.New(),
	}
}

// Read
func (c *Conn) Read(p []byte) (int, error) {
	select {
	case c.readCh <- p:
		n := <-c.sizeCh
		return n, nil
	case <-c.doneCh:
		return 0, io.EOF
	case <-c.readDeadline.Done():
		return 0, context.DeadlineExceeded
	}
}

// Write writes len(p) bytes from p to the DTLS connection
func (c *Conn) Write(p []byte) (n int, err error) {
	select {
	case <-c.writeDeadline.Done():
		return 0, context.DeadlineExceeded
	default:
	}
	return c.listener.pConn.WriteTo(p, c.rAddr)
}

// Close closes the conn and releases any Read calls
func (c *Conn) Close() error {
	var err error
	c.doneOnce.Do(func() {
		c.listener.connWG.Done()
		close(c.doneCh)
		c.listener.connLock.Lock()
		delete(c.listener.conns, c.rAddr.String())
		nConns := len(c.listener.conns)
		c.listener.connLock.Unlock()

		if nConns == 0 && !c.listener.accepting.Load().(bool) {
			// Wait if this is the final connection
			c.listener.readWG.Wait()
			if errClose, ok := c.listener.errClose.Load().(error); ok {
				err = errClose
			}
		} else {
			err = nil
		}
	})

	return err
}

// LocalAddr implements net.Conn.LocalAddr
func (c *Conn) LocalAddr() net.Addr {
	return c.listener.pConn.LocalAddr()
}

// RemoteAddr implements net.Conn.RemoteAddr
func (c *Conn) RemoteAddr() net.Addr {
	return c.rAddr
}

// SetDeadline implements net.Conn.SetDeadline
func (c *Conn) SetDeadline(t time.Time) error {
	c.readDeadline.Set(t)
	c.writeDeadline.Set(t)
	// Deadline of underlying connection should not be changed
	// since the connection can be shared.
	return nil
}

// SetReadDeadline implements net.Conn.SetDeadline
func (c *Conn) SetReadDeadline(t time.Time) error {
	c.readDeadline.Set(t)
	return nil
}

// SetWriteDeadline implements net.Conn.SetDeadline
func (c *Conn) SetWriteDeadline(t time.Time) error {
	c.writeDeadline.Set(t)
	return nil
}
