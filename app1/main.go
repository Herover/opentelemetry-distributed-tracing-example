package main

import (
	"fmt"
	"io"
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
			w.WriteHeader(400)
			fmt.Fprintf(w, "no id")
			return
		}

		requestURL := fmt.Sprintf("http://app2:8080/id=%s", r.URL.Query()["id"][0])
		logger.Printf("rul %s", requestURL)
		res, err := http.Get(requestURL)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "Error: %s", err.Error())
			logger.Printf("request error %s", err.Error())
			return
		}

		logger.Printf("client: status code: %d\n", res.StatusCode)

		body, _ := io.ReadAll(res.Body)

		fmt.Fprint(w, string(body))
	})

	logger.Printf("Starting up")
	http.ListenAndServe(":8080", nil)
}
