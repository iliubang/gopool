package gopool

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

type TcpConn struct {
	conn net.Conn
}

func (tp *TcpConn) Close() error {
	return tp.conn.Close()
}

func (tp *TcpConn) Alive() bool {
	return true
}

func TestCommonPool(t *testing.T) {
	pool, err := NewCommonPool(5, 10, func() (Conn, error) {
		c, e := net.Dial("tcp", "127.0.0.1:3306")
		if e != nil {
			return nil, e
		}
		return &TcpConn{
			conn: c,
		}, nil
	})

	if err != nil {
		t.Error(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(pool Pool, i int) {
			c, e := pool.Acquire()
			if e != nil {
				t.Error(e)
			}
			fmt.Println(i, "do sth.")
			time.Sleep(time.Second)
			pool.Release(c)
			fmt.Println(i, "done.")
			wg.Done()
		}(pool, i)
	}

	wg.Wait()
}