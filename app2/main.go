package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	logger := log.New(os.Stdout, "", 0)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("request %s", r.URL.String())

		if r.Method != "GET" {
			return
		}

		if len(r.URL.Query()["id"]) == 0 {
			fmt.Fprintf(w, "Hi stranger")
		} else {
			fmt.Fprintf(w, "Hi %s", r.URL.Query()["id"][0])
		}
	})

	logger.Printf("Starting up")
	http.ListenAndServe(":8080", nil)
}
