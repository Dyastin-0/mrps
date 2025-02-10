package api

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/Dyastin-0/mrps/internal/common"
	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/Dyastin-0/mrps/internal/health"
	"github.com/Dyastin-0/mrps/internal/logger"
	"github.com/Dyastin-0/mrps/internal/ws"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)

func CORS(next http.Handler) http.Handler {
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

func Auth() http.HandlerFunc {
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
			log.Error().Err(err).Str("type", "access").Str("token", "..."+string(accessToken[len(accessToken)-10:])).Msg("api")
			return
		}

		refreshToken, err := NewToken(expectedEmail, os.Getenv("REFRESH_TOKEN_KEY"), 24*time.Hour)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Error().Err(err).Str("type", "refresh").Str("token", "..."+string(refreshToken[len(refreshToken)-10:])).Msg("api")

			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "rt",
			Value:    refreshToken,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
			MaxAge:   24 * 60 * 60,
			Domain:   os.Getenv("DOMAIN"),
			Path:     "/",
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(accessToken)
	}
}

func JWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token = token[7:]

		claims := &jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("ACCESS_TOKEN_KEY")), nil
		})

		if err != nil {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func Refresh() http.HandlerFunc {
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
			Secure:   true,
			MaxAge:   -1,
			Domain:   config.Misc.Domain,
			Path:     "/",
		})

		claims := &jwt.MapClaims{}
		_, err = jwt.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (interface{}, error) {
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
			Secure:   true,
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
			Type   string               `json:"type"`
			Config common.DomainsConfig `json:"config"`
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
		go config.ParseToYAML()

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

func Signout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     "rt",
			Value:    "",
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
			MaxAge:   -1,
			Domain:   config.Misc.Domain,
			Path:     "/",
		})

		w.WriteHeader(http.StatusOK)
	}
}

func NewToken(email, secret string, expiration time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(expiration).Unix(),
		"iss":   "mrps",
		"iat":   time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

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

func ProtectedRoute() *chi.Mux {
	router := chi.NewRouter()

	router.Use(JWT)

	router.Handle("/", get())
	router.Handle("/uptime", getUptime())
	router.Handle("/health/ws", getHealth())
	router.Handle("/logs/ws", getLogs())
	router.Handle("/{domain}/enabled", setEnabled())

	return router
}
