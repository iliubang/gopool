package gopool

import (
	"errors"
	"io"
	"sync"
)

var (
	ErrParameter = errors.New("parameter error")
	ErrState     = errors.New("pool closed")
)

type Conn interface {
	io.Closer
	Alive() bool
}

type factory func() (Conn, error)

type Pool struct {
	sync.Mutex
	pool      chan Conn
	minConn   int
	maxConn   int
	connCount int
	closed    bool
	factory   factory
}

func NewPool(minConn, maxConn int, factory factory) (*Pool, error) {
	if maxConn < minConn {
		return nil, ErrParameter
	}

	if maxConn <= 0 {
		// TODO use cpu number
		maxConn = 4
	}

	p := &Pool{
		pool:    make(chan Conn, maxConn),
		maxConn: maxConn,
		closed:  false,
		factory: factory,
	}

	for i := 0; i < minConn; i++ {
		conn, err := factory()
		if err != nil {
			continue
		}
		p.connCount++
		p.pool <- conn
	}

	return p, nil
}

func (p *Pool) Acquire() (Conn, error) {
	if p.closed {
		return nil, ErrState
	}

TRY_RESOURCE:
	select {
	case conn := <-p.pool:
		if conn.Alive() {
			return conn, nil
		} else {
			p.Close(conn)
			goto TRY_RESOURCE
		}
	default:
	}

	p.Lock()
	if p.connCount >= p.maxConn {
		p.Unlock()
		conn := <-p.pool
		if conn.Alive() {
			return conn, nil
		} else {
			p.Close(conn)
			goto TRY_RESOURCE
		}
	} else {
		conn, err := p.factory()
		if err != nil {
			p.Unlock()
			return nil, err
		}
		p.connCount++
		p.Unlock()
		return conn, nil
	}
}

func (p *Pool) Release(conn Conn) error {
	if p.closed {
		return ErrState
	}

	p.pool <- conn
	return nil
}

func (p *Pool) Close(conn Conn) error {
	err := conn.Close()
	if err != nil {
		return err
	}
	p.Lock()
	p.connCount--
	p.Unlock()
	return nil
}

func (p *Pool) Shutdown() error {
	if p.closed {
		return nil
	}

	p.Lock()
	close(p.pool)
	for conn := range p.pool {
		conn.Close()
		p.connCount--
	}
	p.closed = true
	p.Unlock()
	return nil
}
