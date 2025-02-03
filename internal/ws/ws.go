package ws

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		for _, allowedOrigin := range config.Misc.AllowedOrigins {
			if r.Header.Get("Origin") == allowedOrigin || allowedOrigin == "*" {
				return true
			}
		}
		return false
	},
}

var Clients = sync.Map{}

func WS(conns ...*sync.Map) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := r.Cookie("rt")
		if err != nil {
			log.Error().Err(err).Msg("Websocket")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		_, err = jwt.ParseWithClaims(token.Value, &jwt.MapClaims{}, func(t *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("REFRESH_TOKEN_KEY")), nil
		})
		if err != nil {
			log.Error().Err(err).Msg("Websocket")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error().Err(err).Msg("Websocket - Upgrader")
			return
		}

		Clients.Store(token.Value, conn)
		log.Info().Str("Status", "connected").Msg("Websocket")

		defer func() {
			for _, cn := range conns {
				cn.Delete(token.Value)
			}
			Clients.Delete(token.Value)
			conn.Close()
		}()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Error().Err(err).Msg("Websocket - Read")
				break
			}
			err = conn.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				log.Error().Err(err).Msg("Websocket - Write")
				break
			}
		}
	}
}

func SendData(id string, data []byte) error {
	if conn, ok := Clients.Load(id); ok {
		if err := conn.(*websocket.Conn).WriteMessage(websocket.TextMessage, data); err != nil {
			return fmt.Errorf("failed to send data: %v", err)
		}
	} else {
		log.Warn().Err(fmt.Errorf("client not found")).Msg("Websocket - Send")
		return fmt.Errorf("client not found")
	}

	return nil
}
