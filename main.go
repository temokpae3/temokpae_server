// Go sever program
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	"github.com/jamespearly/loggly"

	"github.com/gorilla/mux"
)

// Define a struct to store the collected data
type APIData struct {
	InternalName       string `json:"internalName"`
	Title              string `json:"title"`
	MetacriticLink     string `json:"metacriticLink"`
	DealID             string `json:"dealID"`
	StoreID            string `json:"storeID"`
	GameID             string `json:"gameID"`
	SalePrice          string `json:"salePrice"`
	NormalPrice        string `json:"normalPrice"`
	IsOnSale           string `json:"isOnSale"`
	Savings            string `json:"savings"`
	MetacriticScore    string `json:"metacriticScore"`
	SteamRatingText    string `json:"steamRatingText"`
	SteamRatingPercent string `json:"steamRatingPercent"`
	SteamRatingCount   string `json:"steamRatingCount"`
	SteamAppID         string `json:"steamAppID"`
	ReleaseDate        int    `json:"releaseDate"`
	LastChange         int    `json:"lastChange"`
	DealRating         string `json:"dealRating"`
	Thumb              string `json:"thumb"`
}

// Define a struct to store the server time
type resTime struct {
	SystemTime string
}

// Define a struct to store the table status
type TableStatus struct {
	Table string `json:"table"`
	Count *int64 `json:"recordCount"`
}

// Define a struct to store the logging response
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// All Handler function that displays all the items in DynamoDB
func AllHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	client := loggly.New("LOGGLY_TOKEN")
	client.EchoSend("info", "/all endpoint called")
	w.WriteHeader(http.StatusOK)

	// Initialize a session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	if err != nil {
		client.EchoSend("error", "Got error initializing AWS: "+err.Error())
		os.Exit(1)
	}

	// Create DynamoDB client
	svc := dynamodb.New(sess)

	var allResponse []APIData

	// Scan the DB for all the items
	scanErr := svc.ScanPages(&dynamodb.ScanInput{
		TableName: aws.String("test-table-temokpae"),
	}, func(page *dynamodb.ScanOutput, last bool) bool {
		recs := []APIData{}

		err := dynamodbattribute.UnmarshalListOfMaps(page.Items, &recs)
		if err != nil {
			panic(fmt.Sprintf("Failed to unmarshal Dynamodb Scan Items, %v", err))
		}

		allResponse = append(allResponse, recs...)

		return true
	})

	// DB scanning error response
	if scanErr != nil {
		panic(fmt.Sprintf("Got error scanning DB, %v", scanErr))
	}

	// Response of JSON
	json.NewEncoder(w).Encode(allResponse)
}

// Server Handler function that displays the server time
func ServerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	sysTime := resTime{time.Now().String()}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(sysTime)
}

// Status Handler function that displays the table status
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	client := loggly.New("LOGGLY_TOKEN")
	client.EchoSend("info", "/status endpoint called")
	w.WriteHeader(http.StatusOK)

	// Initialize aws session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	if err != nil {
		client.EchoSend("error", "Got error initializing AWS: "+err.Error())
		os.Exit(1)
	}

	// Create DynamoDB client
	svc := dynamodb.New(sess)

	// Describe the table
	input := &dynamodb.DescribeTableInput{
		TableName: aws.String("test-table-temokpae"),
	}

	result, err := svc.DescribeTable(input)
	if err != nil {
		client.EchoSend("error", "Got error describing table: "+err.Error())
		os.Exit(1)
	}

	// Create a response struct to be turned into JSON
	var statusResponse TableStatus
	statusResponse.Table = "test-table-temokpae"
	statusResponse.Count = result.Table.ItemCount

	// JSON Response
	json.NewEncoder(w).Encode(statusResponse)
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
	router.HandleFunc("/temokpae/all", AllHandler).Methods("GET")
	router.HandleFunc("/temokpae/status", StatusHandler).Methods("GET")
	router.HandleFunc("/temokpae/server", ServerHandler).Methods("GET")
	log.Println("Server running...")
	wrappedRouter := logglyMiddleware(router)
	log.Fatal(http.ListenAndServe(":8080", wrappedRouter))
}
