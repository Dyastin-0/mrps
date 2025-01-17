package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Get("/service-1/api/*", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Request URL path: ", r.URL.Path)
		fmt.Fprintf(w, "Hello from service-1")
	})

	fmt.Println("Service 1 is running on port 3002")
	err := http.ListenAndServe(":3002", r)
	if err != nil {
		fmt.Println("Error starting server: ", err)
	}
}
