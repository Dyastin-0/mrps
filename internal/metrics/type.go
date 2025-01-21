package metrics

import "net/http"

type ResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}
