package reverseproxy

import (
	"crypto/tls"
	"io"
	"net"
	"sync"
)

type TCPProxy struct {
	Addr string
	// optional for tls
	WithTLS    bool
	ServerName string
}

func (t *TCPProxy) ForwardTLS(src net.Conn) error {
	tlsconfig := tls.Config{
		ServerName: t.ServerName,
	}
	dst, err := tls.Dial("tcp", t.Addr, &tlsconfig)
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
		errch <- t.stream(src, dst)
		wg.Done()
	}()

	go func() {
		errch <- t.stream(src, dst)
		wg.Done()
	}()

	wg.Wait()
	close(errch)

	for err := range errch {
		if err != nil {
			return err
		}
	}

	return nil
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
		errch <- t.stream(src, dst)
		wg.Done()
	}()

	go func() {
		errch <- t.stream(src, dst)
		wg.Done()
	}()

	wg.Wait()
	close(errch)

	for err := range errch {
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *TCPProxy) stream(src, dst net.Conn) error {
	_, err := io.Copy(dst, src)
	if err != nil {
		return err
	}

	if conn, ok := dst.(*net.TCPConn); ok {
		conn.CloseWrite()
	}

	return nil
}
