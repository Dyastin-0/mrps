package config

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if r.Method == "OPTIONS" {
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
		log.Println("Email:", email)
		log.Println("Password:", password)

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

		accessToken, err := Generate(expectedEmail, os.Getenv("ACCESS_TOKEN_KEY"), 15*time.Minute)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Println("Error generating access token:", err)
			return
		}

		refreshToken, err := Generate(expectedEmail, os.Getenv("REFRESH_TOKEN_KEY"), 24*time.Hour)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Println("Error generating refresh token:", err)
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
			Domain:   os.Getenv("DOMAIN"),
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

		accessToken, err := Generate(email, os.Getenv("ACCESS_TOKEN_KEY"), 15*time.Minute)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		newRefreshToken, err := Generate(email, os.Getenv("REFRESH_TOKEN_KEY"), 24*time.Hour)
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
			Domain:   os.Getenv("DOMAIN"),
			Path:     "/",
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(accessToken)
	}
}

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domains := DomainTrie.GetAll()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(domains)
	})
}

func Generate(email, secret string, expiration time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(expiration).Unix(),
		"iss":   "mrps",
		"iat":   time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}
