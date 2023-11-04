// Go sever program
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/jamespearly/loggly"
	"github.com/microcosm-cc/bluemonday"
	"gopkg.in/validator.v2"

	"github.com/gorilla/mux"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

// Define a struct to store the server time
type resTime struct {
	SystemTime string
}

// Define a struct to store the logging response
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

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

// Define a struct to store the table status
type TableStatus struct {
	Table string `json:"table"`
	Count *int64 `json:"recordCount"`
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

func AllHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	client := loggly.New("LOGGLY_TOKEN")
	client.EchoSend("info", "/all endpoint called")
	w.WriteHeader(http.StatusOK)

	// Initialize a session
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region: aws.String("us-east-1"),
		},
		SharedConfigState: session.SharedConfigEnable,
	}))

	// Create DynamoDB client
	svc := dynamodb.New(sess)

	var allResponse []APIData

	// Table name
	tableName := "test-table-temokpae"

	// Scans the DB for all the items
	scan := svc.ScanPages(&dynamodb.ScanInput{
		TableName: aws.String(tableName),
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
	if scan != nil {
		client.EchoSend("error", "Got error scanning DB: "+scan.Error())
		os.Exit(1)
	}

	// Convert to JSON
	json.NewEncoder(w).Encode(allResponse)
}

// Status Handler function that displays the table status
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	client := loggly.New("LOGGLY_TOKEN")
	client.EchoSend("info", "/status endpoint called")
	w.WriteHeader(http.StatusOK)

	// Initialize aws session
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region: aws.String("us-east-1"),
		},
		SharedConfigState: session.SharedConfigEnable,
	}))

	// Create the DynamoDB client
	svc := dynamodb.New(sess)

	// Table name
	tableName := "test-table-temokpae"

	// Describe the table
	input := &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}

	// Describe table inputed
	result, err := svc.DescribeTable(input)
	if err != nil {
		client.EchoSend("error", "Got error describing table: "+err.Error())
		os.Exit(1)
	}

	// Create a response struct to be turned into JSON
	var statusResponse TableStatus
	statusResponse.Table = "test-table-temokpae"
	statusResponse.Count = result.Table.ItemCount

	// Convert to JSON
	json.NewEncoder(w).Encode(statusResponse)
}

// Search Handler function that searches the table
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	client := loggly.New("LOGGLY_TOKEN")
	client.EchoSend("info", "/search endpoint called")

	// Create a bluemonday policy
	policy := bluemonday.UGCPolicy()

	// Create a query
	query := r.URL.Query()

	// Sanitizes the query parameter
	internalName := policy.Sanitize(query.Get("internalName"))

	// If the query parameter is anything other than internalName, return 400.
	if internalName == "" || len(query) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Regex matching
	matching, err := regexp.MatchString("[a-zA-Z0-9]$", internalName)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		client.EchoSend("error", "Got error matching regex: "+err.Error())
		os.Exit(1)
	}

	// Validating the matching parameter
	if err == validator.Validate(matching) {
		w.WriteHeader(http.StatusBadRequest)
		client.EchoSend("error", "Got error validating: "+err.Error())
		os.Exit(1)
	}

	if matching {
		w.WriteHeader(http.StatusOK)

		// Initialize aws session
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			Config: aws.Config{
				Region: aws.String("us-east-1"),
			},
			SharedConfigState: session.SharedConfigEnable,
		}))

		// Create DynamoDB client
		svc := dynamodb.New(sess)

		// Table name
		tableName := "test-table-temokpae"

		// Create the expression
		filt := expression.Contains(expression.Name("internalName"), internalName)

		expr, err := expression.NewBuilder().WithFilter(filt).Build()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			client.EchoSend("error", "Got error building expression: "+err.Error())
			os.Exit(1)
		}

		// Build the query input parameters
		params := &dynamodb.ScanInput{
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			FilterExpression:          expr.Filter(),
			ProjectionExpression:      expr.Projection(),
			TableName:                 aws.String(tableName),
		}

		// Get all the results of the given internalName
		result, err := svc.Scan(params)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			client.EchoSend("error", "Got error calling Scan: "+err.Error())
			os.Exit(1)
		}

		// Unmarshal the response
		response := []APIData{}
		err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &response)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			client.EchoSend("error", "Got error unmarshalling: "+err.Error())
			os.Exit(1)
		}

		// Convert to JSON
		json.NewEncoder(w).Encode(response)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		message := "Search format: /search?internalName=NAME"
		json.NewEncoder(w).Encode(message)
	}
}

// Implement the http.ResponseWriter interface
func (rw *responseWriter) WriteHeader(status int) {
	rw.statusCode = status
	rw.ResponseWriter.WriteHeader(status)
}

// Implements the http response to bad requests ("POST", "PUT", "DELETE", "PATCH")
func BadRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	client := loggly.New("LOGGLY_TOKEN")
	client.EchoSend("error", "Method: "+r.Method+". Not allowed from: "+r.RemoteAddr+"Path: "+r.RequestURI)
}

// Loggly Middleware function
func logglyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(lrw, r)

		// Tag + client init for Loggly + send message
		client := loggly.New("LOGGLY_TOKEN")
		client.EchoSend("info", "Method type: "+r.Method+" | Source IP address: "+r.RemoteAddr+" | Request Path: "+r.RequestURI+" | Status Code: "+strconv.Itoa(lrw.statusCode))
	})
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/temokpae/server", ServerHandler).Methods("GET")
	router.HandleFunc("/temokpae/all", AllHandler).Methods("GET")
	router.HandleFunc("/temokpae/status", StatusHandler).Methods("GET")
	router.HandleFunc("/temokpae/search", SearchHandler).Queries("internalName", "{internalName:.*}").Methods("GET")
	router.Methods("POST", "PUT", "PATCH", "DELETE").HandlerFunc(BadRequest)
	log.Println("Server running...")
	wrappedRouter := logglyMiddleware(router)
	log.Fatal(http.ListenAndServe(":8080", wrappedRouter))
}
