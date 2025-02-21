package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/Dyastin-0/mrps/internal/health"
	"github.com/Dyastin-0/mrps/internal/logger"
	"github.com/Dyastin-0/mrps/internal/types"
	"github.com/Dyastin-0/mrps/internal/ws"
	sshutil "github.com/Dyastin-0/mrps/pkg/ssh"
	"github.com/go-chi/chi/v5"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)

var sessionCancelMap = make(map[string]context.CancelFunc)

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		allowed := false
		for _, allowedOrigin := range config.Misc.AllowedOrigins {
			if origin == allowedOrigin || allowedOrigin == "*" {
				allowed = true
				break
			}
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func auth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&req)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		email := req.Email
		password := req.Password

		if email == "" || password == "" {
			http.Error(w, "Bad request, missing credentials", http.StatusBadRequest)
			return
		}

		expectedEmail := os.Getenv("AUTH_EMAIL")
		expectedPassword := os.Getenv("AUTH_PASSWORD")

		if email != expectedEmail || password != expectedPassword {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		accessToken, err := NewToken(expectedEmail, os.Getenv("ACCESS_TOKEN_KEY"), 15*time.Minute)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Error().Err(err).Str("type", "access").Str("token", "..."+accessToken[max(0, len(accessToken)-10):]).Msg("api")
			return
		}

		refreshToken, err := NewToken(expectedEmail, os.Getenv("REFRESH_TOKEN_KEY"), 24*time.Hour)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Error().Err(err).Str("type", "refresh").Str("token", "..."+refreshToken[max(0, len(refreshToken)-10):]).Msg("api")

			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "rt",
			Value:    refreshToken,
			HttpOnly: true,
			// SameSite: http.SameSiteNoneMode,
			Secure: config.Misc.Secure,
			MaxAge: 24 * 60 * 60,
			Domain: config.Misc.Domain,
			Path:   "/",
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(accessToken)
	}
}

func jwt(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token = token[7:]

		claims := &jwtv5.MapClaims{}
		_, err := jwtv5.ParseWithClaims(token, claims, func(t *jwtv5.Token) (interface{}, error) {
			return []byte(os.Getenv("ACCESS_TOKEN_KEY")), nil
		})

		if err != nil {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func refresh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("rt")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		refreshToken := cookie.Value

		http.SetCookie(w, &http.Cookie{
			Name:     "rt",
			Value:    "",
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
			Secure:   config.Misc.Secure,
			MaxAge:   -1,
			Domain:   config.Misc.Domain,
			Path:     "/",
		})

		claims := &jwtv5.MapClaims{}
		_, err = jwtv5.ParseWithClaims(refreshToken, claims, func(t *jwtv5.Token) (interface{}, error) {
			return []byte(os.Getenv("REFRESH_TOKEN_KEY")), nil
		})
		if err != nil {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		email, ok := (*claims)["email"].(string)
		if !ok {
			http.Error(w, "Forbidden, invalid token", http.StatusForbidden)
			return
		}

		accessToken, err := NewToken(email, os.Getenv("ACCESS_TOKEN_KEY"), 15*time.Minute)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		newRefreshToken, err := NewToken(email, os.Getenv("REFRESH_TOKEN_KEY"), 24*time.Hour)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "rt",
			Value:    newRefreshToken,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
			Secure:   config.Misc.Secure,
			MaxAge:   24 * 60 * 60,
			Domain:   config.Misc.Domain,
			Path:     "/",
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(accessToken)
	}
}

func setEnabled() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		domain := chi.URLParam(r, "domain")
		token := r.URL.Query().Get("t")

		var req struct {
			Enabled bool `json:"enabled"`
		}

		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&req)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		ok := config.DomainTrie.SetEnabled(domain, req.Enabled)
		if !ok {
			status := "enabled"
			if !req.Enabled {
				status = "disabled"
			}
			http.Error(w, "Domain not modified, it is either not defined or already "+status, http.StatusNotFound)
			return
		}

		con := config.DomainTrie.GetAll()

		conf := struct {
			Type   string              `json:"type"`
			Config types.DomainsConfig `json:"config"`
		}{
			Type:   "config",
			Config: con,
		}

		configBytes, err := json.Marshal(conf)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Error().Err(err).Msg("api")
			return
		}

		go ws.Clients.Send(token, configBytes)

		w.WriteHeader(http.StatusOK)
	}
}

func getHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("t")

		data := struct {
			Type   string                     `json:"type"`
			Health map[string]map[string]bool `json:"health"`
		}{
			Type:   "health",
			Health: config.DomainTrie.GetHealth(),
		}

		health.Subscribers.Store(token, true)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&data)
	}
}

func get() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domains := config.DomainTrie.GetAll()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(domains)
	})
}

func signout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     "rt",
			Value:    "",
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
			Secure:   config.Misc.Secure,
			MaxAge:   -1,
			Domain:   config.Misc.Domain,
			Path:     "/",
		})

		w.WriteHeader(http.StatusOK)
	}
}

func NewToken(email, secret string, expiration time.Duration) (string, error) {
	claims := jwtv5.MapClaims{
		"email": email,
		"exp":   time.Now().Add(expiration).Unix(),
		"iss":   "mrps",
		"iat":   time.Now().Unix(),
	}

	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

func getUptime() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config.StartTime.Unix())
	}
}

func getLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("t")

		readyChan := make(chan bool)

		logger.LeftBehind.Store(token, readyChan)

		go logger.CatchUp(token, readyChan)

		w.WriteHeader(http.StatusOK)

		retry := 20
		ok := false
		for retry > 0 {
			if ok = ws.Clients.Exists(token); ok {
				break
			}
			retry--
			time.Sleep(50 * time.Millisecond)
		}
		readyChan <- ok
	}
}

func sync() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		config.ParseToYAML()
		w.WriteHeader(http.StatusAccepted)
	}
}

func ssh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		token = token[7:]

		conn, _ := ws.Clients.Get(token)
		if conn == nil {
			http.Error(w, "webSocket connection not found", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusAccepted)

		go func() {
			cancel, err := sshutil.StartSession(
				os.Getenv("PRIVATE_KEY"),
				os.Getenv("IP"),
				os.Getenv("HOST_KEY"),
				os.Getenv("USER"),
				token,
				conn,
			)
			if err != nil {
				log.Error().Err(err).Msg("ssh")
			}

			log.Info().Str("status", "connected").Str("client", token[max(0, len(token)-10):]).Msg("ssh")
			sessionCancelMap[token] = cancel
		}()
	}
}

func cancelSSH() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		token = token[7:]

		sessionCancelMap[token]()
		delete(sessionCancelMap, token)

		w.WriteHeader(http.StatusOK)
	}
}

func configRoute() *chi.Mux {
	router := chi.NewRouter()

	router.Use(jwt)

	router.Handle("/", get())
	router.Post("/sync", sync())
	router.Handle("/uptime", getUptime())
	router.Handle("/health/ws", getHealth())
	router.Handle("/logs/ws", getLogs())
	router.Handle("/{domain}/enabled", setEnabled())

	return router
}

func sshRoute() *chi.Mux {
	router := chi.NewRouter()

	router.Use(jwt)

	router.Post("/", ssh())
	router.Delete("/", cancelSSH())

	return router
}

func Start() {
	router := chi.NewRouter()

	router.Use(logger.Handler)
	router.Use(cors)

	router.Mount("/config", configRoute())
	router.Mount("/ssh", sshRoute())

	router.Handle("/refresh", refresh())
	router.Handle("/signout", signout())
	router.Handle("/auth", auth())
	router.Get("/ws", ws.Handler(&health.Subscribers, &logger.Subscribers, &logger.LeftBehind))

	log.Info().Str("status", "running").Str("port", config.Misc.MetricsPort).Msg("api")
	err := http.ListenAndServe(":"+config.Misc.ConfigAPIPort, router)
	if err != nil {
		log.Fatal().Err(err).Msg("api")
	}
}
