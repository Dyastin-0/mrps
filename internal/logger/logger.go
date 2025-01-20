package logger

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func Handler(next http.Handler) http.Handler {
	return middleware.Logger(next)
}
