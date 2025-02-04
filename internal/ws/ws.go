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
		token := r.URL.Query().Get("t")

		if token == "" {
			log.Error().Err(fmt.Errorf("unauthorized")).Msg("Websocket")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		_, err := jwt.ParseWithClaims(token, &jwt.MapClaims{}, func(t *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("ACCESS_TOKEN_KEY")), nil
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

		Clients.Store(token, conn)
		log.Info().Str("Status", "connected").Msg("Websocket")

		defer func() {
			for _, cn := range conns {
				cn.Delete(token)
			}
			Clients.Delete(token)
			conn.Close()
		}()

		for {
			messageType, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.Error().Err(err).Msg("Websocket - Unexpected disconnect")
				} else {
					log.Info().Msg("Websocket - Client disconnected")
				}
				break
			}

			if err := conn.WriteMessage(messageType, msg); err != nil {
				log.Error().Err(err).Msg("Websocket - Write failed")
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
