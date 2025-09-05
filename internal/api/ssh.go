package api

import (
	"context"
	"net/http"
	"os"
	"time"

	sshutil "github.com/Dyastin-0/mrps/internal/ssh"
	"github.com/Dyastin-0/mrps/internal/ws"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

var sessionCancelMap = make(map[string]context.CancelFunc)

func ssh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		token = token[7:]

		retry := 20
		ok := false
		for retry > 0 {
			if ok = ws.Clients.Exists(token); ok {
				break
			}
			retry--
			time.Sleep(50 * time.Millisecond)
		}

		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		conn, _ := ws.Clients.Get(token)

		cred := &sshutil.SessionCredentials{
			PrivateKey: os.Getenv("PRIVATE_KEY"),
			InstanceIP: os.Getenv("IP"),
			HostKey:    os.Getenv("HOST_KEY"),
			User:       os.Getenv("USER"),
		}

		cancel, err := sshutil.StartSession(
			cred,
			token,
			conn,
		)
		if err != nil {
			log.Error().Err(err).Msg("ssh")
			w.WriteHeader(http.StatusInternalServerError)
		}

		log.Info().Str("status", "connected").Str("client", "..."+token[max(0, len(token)-10):]).Msg("ssh")
		sessionCancelMap[token] = cancel

		w.WriteHeader(http.StatusOK)
	}
}

func cancelSSH() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		token = token[7:]

		cancel := sessionCancelMap[token]

		if cancel != nil {
			cancel()
			delete(sessionCancelMap, token)
		}

		w.WriteHeader(http.StatusOK)
	}
}

func sshRoute() *chi.Mux {
	router := chi.NewRouter()

	router.Use(jwt)

	router.Post("/", ssh())
	router.Delete("/", cancelSSH())

	return router
}
