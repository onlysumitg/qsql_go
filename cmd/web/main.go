package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/onlysumitg/qsql2/internal/models"
	bolt "go.etcd.io/bbolt"
)

func main() {
	fmt.Printf("Starting app")

	// go run ./cmd/web -post=4002 -host="localhost"
	// go run ./cmd/web -h  ==> help text
	// default value for addr => ":4000"

	// using single var
	// addr := flag.String("addr", ":4000", "HTTP work addess")
	// fmt.Printf("\nStarting servers at port %s", *addr)
	// err := http.ListenAndServe(*addr, getTestRoutes())

	//using struct

	//--------------------------------------- Setup CLI paramters ----------------------------
	var params parameters
	flag.StringVar(&params.host, "host", "127.0.0.1", "Http Host Name")
	flag.IntVar(&params.port, "port", 4040, "Port")
	flag.Parse()

	// --------------------------------------- Setup database ----------------------------
	db, err := bolt.Open("db/internal.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// --------------------------------------- Setup app config and dependency injection ----------------------------
	app := baseAppConfig(params, db)
	routes := app.routes()
	app.batches()
	//--------------------------------------- Setup websockets ----------------------------
	go ListenToWsChannel()

	go models.LoadQueryMap(app.shorthandQueries, app.savedQueries)

	fmt.Printf("\nStarting servers at port %s \n", params.getHttpAddress())

	// this is short cut to create http.Server and  server.ListenAndServe()
	// err := http.ListenAndServe(params.addr, routes)

	server := &http.Server{
		Addr:     params.getHttpAddress(),
		Handler:  routes,
		ErrorLog: app.errorLog,
	}

	// url := "http://localhost" + params.addr
	// go openbrowser(url)
	err = server.ListenAndServe()
	log.Fatal(err)

	// mux := http.NewServeMux()
	// mux.Handle("/", http.HandlerFunc(home))
}
