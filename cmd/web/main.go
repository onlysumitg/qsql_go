package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	"github.com/onlysumitg/qsql2/internal/models"

	bolt "go.etcd.io/bbolt"
)

type parameters struct {
	addr string
	//staticDir string
	//flag      bool
}

type application struct {
	errorLog       *log.Logger
	infoLog        *log.Logger
	templateCache  map[string]*template.Template
	formDecoder    *form.Decoder
	sessionManager *scs.SessionManager
	users          *models.UserModel
	servers        *models.ServerModel
	savedQueries   *models.SavedQueryModel
	batchSQLModel  *models.BatchSQLModel
	InProduction   bool
}

func main() {
	fmt.Printf("Starting app")

	// go run ./cmd/web -addr=":4002"
	// go run ./cmd/web -h  ==> help text
	// default value for addr => ":4000"

	// using single var
	// addr := flag.String("addr", ":4000", "HTTP work addess")
	// fmt.Printf("\nStarting servers at port %s", *addr)
	// err := http.ListenAndServe(*addr, getTestRoutes())

	//using struct
	var params parameters
	flag.StringVar(&params.addr, "addr", ":4000", "HTTP work addess")

	flag.Parse()

	infoLog := log.New(os.Stderr, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	fmt.Printf("Setting db")
	// setup data base
	db, err := bolt.Open("internal.db", 0600, nil)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer db.Close()

	// Dependency injection

	// Use the scs.New() function to initialize a new session manager. Then we
	// configure it to use our MySQL database as the session store, and set a
	// lifetime of 12 hours (so that sessions automatically expire 12 hours
	// after first being created).
	fmt.Printf("Setting session")

	sessionManager := scs.New()
	sessionManager.Lifetime = 12 * time.Hour
	sessionManager.Cookie.Persist = true
	sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	//sessionManager.Cookie.Secure = app.InProduction

	templateCache, err := newTemplateCache()
	if err != nil {
		errorLog.Fatal(err)
	}

	formDecoder := form.NewDecoder()

	app := application{
		errorLog:       errorLog,
		infoLog:        infoLog,
		templateCache:  templateCache,
		sessionManager: sessionManager,
		formDecoder:    formDecoder,
		users:          &models.UserModel{DB: db},
		servers:        &models.ServerModel{DB: db},
		savedQueries:   &models.SavedQueryModel{DB: db},
		batchSQLModel:  &models.BatchSQLModel{DB: db},
	}

	routes := app.routes()
	app.batches()

	fmt.Printf("\nStarting servers at port %s", params.addr)

	// this is short cut to create http.Server and  server.ListenAndServe()
	// err := http.ListenAndServe(params.addr, routes)

	server := &http.Server{
		Addr:     params.addr,
		Handler:  routes,
		ErrorLog: errorLog,
	}

	// url := "http://localhost" + params.addr
	// go openbrowser(url)
	err = server.ListenAndServe()
	log.Fatal(err)
}

func openbrowser(url string) {
	log.Println("Opening browser:", url)
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}

}
