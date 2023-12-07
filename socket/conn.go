package socket

import (
	"bytes"
	"net"
	"sync"
	"time"
)

type ConnWrap struct {
	conn   net.Conn
	notify chan int

	mutex  sync.Mutex
	err    error
	buffer bytes.Buffer
}

func New(conn net.Conn) *ConnWrap {
	pwrap := new(ConnWrap)
	pwrap.conn = conn
	pwrap.notify = make(chan int, 1)
	go pwrap.loopSend()
	return pwrap
}

func (wrap *ConnWrap) Read(b []byte) (n int, err error) {
	return wrap.conn.Read(b)
}

func (wrap *ConnWrap) Write(b []byte) (n int, err error) {
	wrap.mutex.Lock()
	if wrap.err != nil {
		wrap.mutex.Unlock()
		return 0, wrap.err
	}
	wrap.buffer.Write(b)
	wrap.mutex.Unlock()
	select {
	case wrap.notify <- 1:
	default:
	}
	return len(b), nil
}

func doWrite(conn net.Conn, data []byte) error {
	writed := 0
	for writed < len(data) {
		n, err := conn.Write(data[writed:])
		if err != nil {
			return err
		}
		writed += n
	}
	return nil
}

func (wrap *ConnWrap) loopSend() {
	for {
		<-wrap.notify
		wrap.mutex.Lock()
		tmp := make([]byte, wrap.buffer.Len())
		copy(tmp, wrap.buffer.Bytes())
		wrap.buffer.Reset()
		wrap.mutex.Unlock()
		err := doWrite(wrap.conn, tmp)
		if err != nil {
			wrap.mutex.Lock()
			wrap.err = err
			wrap.mutex.Unlock()
			break
		}
	}
}

func (wrap *ConnWrap) Close() error {
	return wrap.conn.Close()
}

func (wrap *ConnWrap) LocalAddr() net.Addr {
	return wrap.conn.LocalAddr()
}

func (wrap *ConnWrap) RemoteAddr() net.Addr {
	return wrap.conn.RemoteAddr()
}

func (wrap *ConnWrap) SetDeadline(t time.Time) error {
	return wrap.conn.SetDeadline(t)
}

func (wrap *ConnWrap) SetReadDeadline(t time.Time) error {
	return wrap.conn.SetReadDeadline(t)
}

func (wrap *ConnWrap) SetWriteDeadline(t time.Time) error {
	return wrap.conn.SetWriteDeadline(t)
}
