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

var Clients = NewHub()

func Handler(conns ...*sync.Map) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("t")

		if token == "" {
			log.Error().Err(fmt.Errorf("unauthorized")).Msg("websocket")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		_, err := jwt.ParseWithClaims(token, &jwt.MapClaims{}, func(t *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("ACCESS_TOKEN_KEY")), nil
		})
		if err != nil {
			log.Error().Err(err).Msg("websocket")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error().Err(err).Msg("Websocket upgrade failed")
			return
		}

		closed := make(chan bool)
		Clients.register <- connection{id: token, conn: conn, closed: closed}

		shortToken := "..." + token[max(0, len(token)-10):]
		log.Info().Str("status", "connected").Str("client", shortToken).Msg("websocket")

		defer func() {
			for _, cn := range conns {
				cn.Delete(token)
			}
			Clients.unregister <- token
			conn.Close()
			close(closed)
		}()

		<-closed
		log.Info().Str("status", "closed").Str("client", shortToken).Msg("websocket")

	}
}
