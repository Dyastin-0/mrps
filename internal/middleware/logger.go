package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func Logger(next http.Handler) http.Handler {
	return middleware.Logger(next)
}
