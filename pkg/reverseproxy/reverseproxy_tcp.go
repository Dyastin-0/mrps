package reverseproxy

import (
	"io"
	"net"

	"github.com/rs/zerolog/log"
)

type TCPProxy struct {
	Addr       string
	ServerName string
}

func (t *TCPProxy) Forward(src net.Conn) error {
	log.Logger.Debug().Str("addr", t.Addr).Msg("forward hit")
	dst, err := net.Dial("tcp", t.Addr)
	if err != nil {
		src.Close()
		return err
	}

	go t.stream(src, dst)
	go t.stream(dst, src)

	return nil
}

func (t *TCPProxy) stream(dst, src net.Conn) {
	defer dst.Close()
	defer src.Close()

	log.Info().Msg("stream hit")

	io.Copy(dst, src)
}
