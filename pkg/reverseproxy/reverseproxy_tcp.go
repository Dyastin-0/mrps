package reverseproxy

import (
	"io"
	"net"
	"sync"
)

type TCPProxy struct {
	Addr       string
	ServerName string
}

func (t *TCPProxy) Forward(src net.Conn) error {
	dst, err := net.Dial("tcp", t.Addr)
	if err != nil {
		src.Close()
		return err
	}

	defer func() {
		src.Close()
		dst.Close()
	}()

	errch := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		t.stream(&wg, src, dst, errch)
	}()

	go func() {
		t.stream(&wg, dst, src, errch)
	}()

	go func() {
		wg.Wait()
		close(errch)
	}()

	for err := range errch {
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *TCPProxy) stream(wg *sync.WaitGroup, src, dst net.Conn, errch chan error) {
	defer wg.Done()

	_, err := io.Copy(dst, src)
	if err != nil {
		errch <- err
	}

	if conn, ok := dst.(*net.TCPConn); ok {
		conn.CloseWrite()
	}
}
