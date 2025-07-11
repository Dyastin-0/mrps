package tls

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/Dyastin-0/mrps/internal/types"
	"github.com/caddyserver/certmagic"
	"github.com/rs/zerolog/log"
)

type TLS struct {
	addr, domain string
	cancel       context.CancelFunc
}

func New(addr, domain string) *TLS {
	return &TLS{
		addr:   addr,
		domain: domain,
	}
}

func (t *TLS) StopHealthChecks() {
	if t.cancel != nil {
		t.cancel()
	}
}

func (t *TLS) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)

	t.cancel = cancel

	magic := certmagic.NewDefault()

	err := magic.ManageSync(ctx, []string{t.domain})
	if err != nil {
		return err
	}

	ln, err := tls.Listen("tcp", t.addr, magic.TLSConfig())
	if err != nil {
		return err
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				log.Info().Str("context", "cancelled").Msg("tcp")
				return nil
			default:
				log.Err(err).Msg("tcp")
				continue
			}
		}

		go func() {
			err := t.handleConn(conn)
			if err != nil {
				log.Error().Err(err).Msg("tcp handleConn")
			}
		}()
	}
}

func (t *TLS) handleConn(conn net.Conn) error {
	sni := getSNI(conn)
	config := config.DomainTrie.MatchWithProto(sni, types.TCPProtocol)

	log.Debug().Str("sni", sni).Msg("SNI")

	route := config.Routes["/"]

	if route.BalancerType != "" {
		route.BalancerTCP.Serve(conn)
	} else {
		dst := route.BalancerTCP.First()
		log.Debug().Str("AT", "first").Msg("TCP")
		err := dst.ProxyTCP.Forward(conn)
		log.Debug().Str("AT", "forward").Msg("TCP")
		if err != nil {
			log.Error().Err(err).Msg("tcp err")
		}
	}

	return nil
}

func getSNI(conn net.Conn) string {
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return ""
	}

	if err := tlsConn.Handshake(); err != nil {
		log.Warn().Err(err).Msg("tls handshake failed")
		return ""
	}

	state := tlsConn.ConnectionState()
	return state.ServerName
}
