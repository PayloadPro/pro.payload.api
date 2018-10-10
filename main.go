package main

import (
	"database/sql"
	"flag"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"golang.org/x/net/context"

	"github.com/PayloadPro/api/configs"
	"github.com/PayloadPro/api/deps"
	"github.com/PayloadPro/api/rpc"
	"github.com/PayloadPro/api/services"
)

func main() {

	var err error

	sa := getFlagConfig()

	// Services
	services := &deps.Services{
		Bin:     &services.BinService{},
		Request: &services.RequestService{},
		Stats:   &services.StatsService{},
	}

	// Config
	config := &deps.Config{
		App: &configs.AppConfig{},
		DB:  &configs.CockroachConfig{},
	}
	config.Setup()

	// Create a DB Connection
	db, err := sql.Open("postgres", config.DB.DSN)
	if err != nil {
		log.Fatal("error connecting to the database: ", err)
	}

	defer db.Close()

	// Add the DB to the Service
	services.Bin.DB = db
	services.Request.DB = db
	services.Stats.DB = db

	router := createRouter(services, config)

	log.Fatal(http.ListenAndServe(*sa, handler(router)))
}

func handler(router *mux.Router) http.Handler {
	origins := handlers.AllowedOrigins([]string{"X-Requested-With", "Content-Type", "Authorization"})
	headers := handlers.AllowedHeaders([]string{"*"})
	methods := handlers.AllowedMethods([]string{"DELETE", "GET", "HEAD", "PATCH", "POST", "PUT", "OPTIONS"})

	return handlers.CORS(origins, headers, methods)(router)
}

func createRouter(services *deps.Services, config *deps.Config) *mux.Router {

	// Context
	rand.Seed(time.Now().UnixNano())
	root := context.Background()
	ctx, cancel := context.WithCancel(root)
	defer cancel()

	router := mux.NewRouter()

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		JSONEndpointHandler(w, r, func() (interface{}, int, error) {
			return rpc.NewGetRoot(services, config)(ctx, r)
		})
	}).Methods("GET")

	router.HandleFunc("/bins", func(w http.ResponseWriter, r *http.Request) {
		JSONEndpointHandler(w, r, func() (interface{}, int, error) {
			return rpc.NewCreateBin(services, config)(ctx, r)
		})
	}).Methods("POST")

	router.HandleFunc("/bins", func(w http.ResponseWriter, r *http.Request) {
		JSONEndpointHandler(w, r, func() (interface{}, int, error) {
			return rpc.NewGetBins(services, config)(ctx, r)
		})
	}).Methods("GET")

	router.HandleFunc("/bins/{id}", func(w http.ResponseWriter, r *http.Request) {
		JSONEndpointHandler(w, r, func() (interface{}, int, error) {
			return rpc.NewGetBin(services, config)(ctx, r)
		})
	}).Methods("GET")

	router.HandleFunc("/bins/{id}/request", func(w http.ResponseWriter, r *http.Request) {
		JSONEndpointHandler(w, r, func() (interface{}, int, error) {
			return rpc.NewCreateRequest(services, config)(ctx, r)
		})
	})

	router.HandleFunc("/bins/{id}/requests", func(w http.ResponseWriter, r *http.Request) {
		JSONEndpointHandler(w, r, func() (interface{}, int, error) {
			return rpc.NewGetRequestsForBin(services, config)(ctx, r)
		})
	}).Methods("GET")

	router.HandleFunc("/bins/{id}/requests/{request_id}", func(w http.ResponseWriter, r *http.Request) {
		JSONEndpointHandler(w, r, func() (interface{}, int, error) {
			return rpc.NewGetRequestForBin(services, config)(ctx, r)
		})
	}).Methods("GET")

	return router
}

// getFlagConfig sets the runtime variables
func getFlagConfig() *string {

	fs := flag.NewFlagSet("", flag.ExitOnError)
	server := fs.String("server", "0.0.0.0", "HTTP server")
	port := fs.String("port", "8081", "HTTP server port")
	flag.Usage = fs.Usage

	fs.Parse(os.Args[1:])

	si := make([]string, 0)
	si = append(si, *server, *port)

	sa := strings.Join(si, ":")

	return &sa
}
