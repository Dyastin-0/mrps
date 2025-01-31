package config

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		for _, allowedOrigin := range Misc.AllowedOrigins {
			if r.Header.Get("Origin") == allowedOrigin || allowedOrigin == "*" {
				return true
			} else {
			}
		}
		return false
	},
}

var WSClients = sync.Map{}

func WS(w http.ResponseWriter, r *http.Request) {
	token, err := r.Cookie("rt")
	if err != nil {
		log.Println("Failed to get token:", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	_, err = jwt.ParseWithClaims(token.Value, &jwt.MapClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("REFRESH_TOKEN_KEY")), nil
	})
	if err != nil {
		log.Println("Failed to get token:", err)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		return
	}

	WSClients.Store(token.Value, conn)
	log.Println("Client connected:", token.Value)

	defer func() {
		WSClients.Delete(token.Value)
		HealthSubscribers.Delete(token.Value)
		conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}
		err = conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("Write error:", err)
			break
		}
	}
}

func SendData(id string, data []byte) error {
	if conn, ok := WSClients.Load(id); ok {
		if err := conn.(*websocket.Conn).WriteMessage(websocket.TextMessage, data); err != nil {
			WSClients.Delete(id)
			return fmt.Errorf("failed to send data: %v", err)
		}
	} else {
		log.Println("Client not found:", id)
		return fmt.Errorf("client not found")
	}

	return nil
}
