// Go sever program
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jamespearly/loggly"

	"github.com/gorilla/mux"
)

// Define a struct to store the server time
type resTime struct {
	SystemTime string
}

// Define a struct to store the logging response
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// Server Handler function that displays the server time
func ServerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	client := loggly.New("LOGGLY_TOKEN")
	client.EchoSend("info", "/server endpoint called")
	w.WriteHeader(http.StatusOK)
	sysTime := resTime{time.Now().String()}
	json.NewEncoder(w).Encode(sysTime)
}

// Implement the http.ResponseWriter interface
func NewLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

// Loggly Middleware function
func logglyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := NewLoggingResponseWriter(w)
		next.ServeHTTP(lrw, r)

		// Tag + client init for Loggly + send message
		client := loggly.New("LOGGLY_TOKEN")
		client.EchoSend("info", "Method type: "+r.Method+" | Source IP address: "+r.RemoteAddr+" | Request Path: "+r.RequestURI+" | Status Code: "+strconv.Itoa(lrw.statusCode))
	})
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/temokpae/server", ServerHandler).Methods("GET")
	log.Println("Server running...")
	wrappedRouter := logglyMiddleware(router)
	log.Fatal(http.ListenAndServe(":8080", wrappedRouter))
}
