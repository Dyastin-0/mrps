package iphash

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/Dyastin-0/mrps/internal/types"
	"github.com/stretchr/testify/assert"
)

func startTCPEchoServer(addr string) {
	go func() {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			panic(err)
		}
		for {
			conn, err := ln.Accept()
			if err != nil {
				continue
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				n, _ := c.Read(buf)
				c.Write([]byte("echo: " + string(buf[:n])))
			}(conn)
		}
	}()
	time.Sleep(300 * time.Millisecond)
}

type fakeConn struct {
	net.Conn
}

func (f fakeConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("192.168.1.99"), Port: 12345}
}

func TestIPHashTCPForward(t *testing.T) {
	startTCPEchoServer("127.0.0.1:9001")
	startTCPEchoServer("127.0.0.1:9002")

	dests := []types.Dest{
		{URL: "127.0.0.1:9001"},
		{URL: "127.0.0.1:9002"},
	}

	balancer := NewTCP(context.Background(), dests, 1000*time.Millisecond)

	clientConn, serverConn := net.Pipe()

	go func() {
		_, err := clientConn.Write([]byte("hello"))
		assert.NoError(t, err)

		reply := make([]byte, 1024)
		n, _ := clientConn.Read(reply)
		assert.Contains(t, string(reply[:n]), "echo: hello")
		clientConn.Close()
	}()

	ok := balancer.Serve(fakeConn{serverConn}, "")
	assert.True(t, ok, "Serve should return true")
}
