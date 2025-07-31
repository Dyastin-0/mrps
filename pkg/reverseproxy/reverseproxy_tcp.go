package reverseproxy

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/rs/zerolog/log"
)

type TCPProxy struct {
	Addr string
	// optional for tls
	WithTLS bool
}

func (t *TCPProxy) ForwardTLS(dst net.Conn, sni string) error {
	if sni == "" {
		return fmt.Errorf("tls missing sni")
	}

	log.Debug().Msg("F")

	tlsconfig := &tls.Config{
		ServerName: sni,
	}

	src, err := tls.Dial("tcp", t.Addr, tlsconfig)
	if err != nil {
		dst.Close()
		return fmt.Errorf("failed to dial tls: %v", err)
	}

	state := src.ConnectionState()
	log.Debug().Msg("ServerName: " + state.ServerName)

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
		errch <- t.stream(dst, src)
		wg.Done()
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

func (t *TCPProxy) Forward(dst net.Conn) error {
	src, err := net.Dial("tcp", t.Addr)
	if err != nil {
		dst.Close()
		return err
	}

	log.Debug().Msg("E")

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
		errch <- t.stream(dst, src)
		wg.Done()
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
