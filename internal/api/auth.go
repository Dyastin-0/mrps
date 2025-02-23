package api

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/Dyastin-0/mrps/internal/health"
	"github.com/Dyastin-0/mrps/internal/logger"
	"github.com/Dyastin-0/mrps/internal/ws"
	"github.com/go-chi/chi/v5"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)

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

func auth(w http.ResponseWriter, r *http.Request) {
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

	accessToken, err := newToken(expectedEmail, os.Getenv("ACCESS_TOKEN_KEY"), 15*time.Minute)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Error().Err(err).Str("type", "access").Str("token", "..."+accessToken[max(0, len(accessToken)-10):]).Msg("api")
		return
	}

	refreshToken, err := newToken(expectedEmail, os.Getenv("REFRESH_TOKEN_KEY"), 24*time.Hour)
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

func refresh(w http.ResponseWriter, r *http.Request) {
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

	accessToken, err := newToken(email, os.Getenv("ACCESS_TOKEN_KEY"), 15*time.Minute)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	newRefreshToken, err := newToken(email, os.Getenv("REFRESH_TOKEN_KEY"), 24*time.Hour)
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

func signout(w http.ResponseWriter, r *http.Request) {
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

func newToken(email, secret string, expiration time.Duration) (string, error) {
	claims := jwtv5.MapClaims{
		"email": email,
		"exp":   time.Now().Add(expiration).Unix(),
		"iss":   "mrps",
		"iat":   time.Now().Unix(),
	}

	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

func authRoute() *chi.Mux {
	router := chi.NewRouter()

	router.Post("/refresh", refresh)
	router.Post("/signout", signout)
	router.Post("/auth", auth)
	router.Get("/ws", ws.Handler(&health.Subscribers, &logger.Subscribers, &logger.LeftBehind))

	return router
}
