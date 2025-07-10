package reverseproxy

import (
	"io"
	"net"
)

type TCPProxy struct {
	Addr       string
	ServerName string
}

func (t *TCPProxy) Forward(src net.Conn) error {
	dst, err := net.Dial("tcp", t.Addr)
	if err != nil {
		return err
	}

	go t.stream(src, dst)
	go t.stream(dst, src)

	return nil
}

func (t *TCPProxy) stream(dst, src net.Conn) {
	defer dst.Close()
	defer src.Close()
	io.Copy(dst, src)
}
