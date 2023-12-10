package main

import (
	"earthworm/socket"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

var mutex sync.Mutex
var gser *Server

func GetServer() *Server {
	mutex.Lock()
	tmp := gser
	mutex.Unlock()
	return tmp
}

func SetServer(ser *Server) {
	mutex.Lock()
	tmp := gser
	gser = ser
	mutex.Unlock()
	if tmp != nil {
		tmp.CloseClientConn()
	}
}

func main() {
	password := flag.String("password", "", "auth password")
	client := flag.String("client", "", "client listen address")
	socks5 := flag.String("socks5", "", "socks5 listen address")
	flag.Parse()
	if len(*password) == 0 || len(*client) == 0 || len(*socks5) == 0 {
		flag.Usage()
		return
	}
	clientListen, err := net.Listen("tcp", *client)
	if err != nil {
		fmt.Println(err)
		return
	}
	socks5Listen, err := net.Listen("tcp", *socks5)
	if err != nil {
		fmt.Println(err)
		return
	}
	go socks5Accept(socks5Listen)
	clientAccept(clientListen, *password)
}

func clientAccept(clientListen net.Listener, password string) {
	for {
		conn, err := clientListen.Accept()
		if err != nil {
			continue
		}
		go clientRead(socket.New(conn), password)
	}
}

func clientRead(conn *socket.ConnWrap, password string) {
	defer conn.Close()
	tmp := make([]byte, len(password))
	if _, err := io.ReadFull(conn, tmp); err != nil {
		return
	}
	if password != string(tmp) {
		return
	}
	fmt.Println("client come back")
	pser := NewServer(conn)
	SetServer(pser)
	go clientHeartBeat(conn)
	for {
		var head [5]byte
		if _, err := io.ReadFull(conn, head[:]); err != nil {
			fmt.Printf("read client head fail:%s\n", err.Error())
			break
		}
		bodyLen := binary.LittleEndian.Uint16(head[:])
		sid := binary.LittleEndian.Uint16(head[2:])
		op := head[4]
		bodyData := make([]byte, bodyLen)
		if bodyLen != 0 {
			if _, err := io.ReadFull(conn, bodyData); err != nil {
				fmt.Printf("read client body fail:%s\n", err.Error())
				break
			}
		}
		pser.ClientMsg(int(sid), op, bodyData)
	}
	pser.Stop()
}

func clientHeartBeat(cli *socket.ConnWrap) {
	hb := socket.MakeMsg(0, socket.HEARTBEAT, nil)
	for {
		time.Sleep(4 * time.Second)
		_, err := cli.Write(hb)
		if err != nil {
			break
		}
	}
}

func socks5Accept(socks5Listen net.Listener) {
	for {
		conn, err := socks5Listen.Accept()
		if err != nil {
			continue
		}
		go socks5Read(socket.New(conn))
	}
}

func socks5Read(conn *socket.ConnWrap) {
	defer conn.Close()
	var handHead [2]byte
	if _, err := io.ReadFull(conn, handHead[:]); err != nil {
		return
	}
	if handHead[0] != 5 || handHead[1] == 0 {
		return
	}
	handBody := make([]byte, handHead[1])
	if _, err := io.ReadFull(conn, handBody); err != nil {
		return
	}
	conn.Write([]byte{5, 0})
	var reqHead [4]byte
	if _, err := io.ReadFull(conn, reqHead[:]); err != nil {
		return
	}
	if reqHead[0] != 5 || reqHead[1] != 1 || reqHead[2] != 0 {
		return
	}
	if reqHead[3] != 1 && reqHead[3] != 3 {
		return
	}
	var reqBody []byte
	if reqHead[3] == 1 {
		var tmp [6]byte
		if _, err := io.ReadFull(conn, tmp[:]); err != nil {
			return
		}
		reqBody = append(reqBody, 1)
		reqBody = append(reqBody, tmp[:]...)
	} else {
		var length [1]byte
		if _, err := io.ReadFull(conn, length[:]); err != nil {
			return
		}
		if length[0] == 0 {
			return
		}
		tmp := make([]byte, length[0]+2)
		if _, err := io.ReadFull(conn, tmp); err != nil {
			return
		}
		reqBody = append(reqBody, 3, length[0])
		reqBody = append(reqBody, tmp...)
	}
	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	pser := GetServer()
	if pser == nil {
		return
	}
	sid := pser.AddSession(conn)
	if sid == 0 {
		return
	}
	pser.ClientWrite(socket.MakeMsg(uint16(sid), socket.CONNECT, reqBody))
	var buffer [1024]byte
	for {
		n, err := conn.Read(buffer[:])
		if err != nil {
			break
		}
		if n == 0 {
			continue
		}
		datamsg := socket.MakeMsg(uint16(sid), socket.DATA, buffer[:n])
		pser.ClientWrite(datamsg)
	}
	time.Sleep(20 * time.Second)
	pser.DeleSession(sid)
}
