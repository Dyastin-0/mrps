package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Get("/service/api/*", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Request URL: ", r.URL)
		fmt.Fprintf(w, "Hello from service")
	})

	fmt.Println("Service is running on port 3001")
	err := http.ListenAndServe(":3001", r)
	if err != nil {
		fmt.Println("Error starting server: ", err)
	}
}
