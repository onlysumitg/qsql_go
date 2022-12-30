package main

// import "net/http"

// func (app *application) testRoutes() *http.ServeMux {
// 	// http.HandleFunc(xx,yy) => this also use a pre built default ServerMux -->  var DefaultServeMux = &defaultServeMux
// 	mux := http.NewServeMux()

// 	// Test handlers
// 	mux.HandleFunc("/helloworld", app.helloworld)  // app route
// 	mux.HandleFunc("/template", templates)   // independent route

// 	// file downloader
// 	mux.HandleFunc("/download", downloadFileHandler)

// 	// static files => http://127.0.0.1:4000/static/
// 	fileServer := http.FileServer(http.Dir("./ui/static/"))

// 	// http.StripPrefix is a middle ware
// 	mux.Handle("/static/", http.StripPrefix("/static", fileServer))

// 	return mux
// }

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/onlysumitg/qsql2/ui" // New import
)

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func addMiddleWares(app *application, router *chi.Mux) {

	// session middleware
	router.Use(app.sessionManager.LoadAndSave)

	// A good base middleware stack : inbuilt in chi
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Use(middleware.Heartbeat("/ping"))

	// CSRF
	router.Use(noSurf)
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func addStaticFiles(router *chi.Mux) {
	// Take the ui.Files embedded filesystem and convert it to a http.FS type so
	// that it satisfies the http.FileSystem interface. We then pass that to the
	// http.FileServer() function to create the file server handler.
	fileServer := http.FileServer(http.FS(ui.Files))

	// Our static files are contained in the "static" folder of the ui.Files
	// embedded filesystem. So, for example, our CSS stylesheet is located at
	// "static/css/main.css". This means that we now longer need to strip the
	// prefix from the request URL -- any requests that start with /static/ can
	// just be passed directly to the file server and the corresponding static
	// file will be served (so long as it exists).
	// router.Handle("/static/*", http.StripPrefix("/static", fileServer))
	router.Handle("/static/*", fileServer)
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func addDummyRoutes(app *application, router *chi.Mux) {
	router.Get("/helloworld", app.helloworld)
	router.Get("/advaancetemplate", app.templatesAdvance)
	router.Get("/download", downloadFileHandler)
	router.Get("/template", templates)
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (app *application) routes() http.Handler {
	router := chi.NewRouter()

	// for more ideas, see: https://developer.github.com/v3/#cross-origin-resource-sharing
	router.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"https://*", "http://*"},

		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},

		ExposedHeaders: []string{"Link"},

		AllowCredentials: false,

		MaxAge: 300, // Maximum value not ignored by any of major browsers
	}))

	addMiddleWares(app, router)
	addStaticFiles(router)

	addDummyRoutes(app, router)
	router.Get("/", app.langingPage)
	router.Get("/help", app.helpPage)

	app.ServerHandlers(router)
	app.QueryHandlers(router)
	app.SavedQueryHandlers(router)
	app.BatchQueryHandlers(router)
	app.WsHandlers(router)
	app.ShorthandQueryHandlers(router)

	// router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	app.notFound(w)
	// })

	// dynamic := alice.New(app.sessionManager.LoadAndSave, noSurf, app.authenticate)

	// router.Handler(http.MethodGet, "/", dynamic.ThenFunc(app.home))
	// router.Handler(http.MethodGet, "/snippet/view/:id", dynamic.ThenFunc(app.snippetView))
	// router.Handler(http.MethodGet, "/user/signup", dynamic.ThenFunc(app.userSignup))
	// router.Handler(http.MethodPost, "/user/signup", dynamic.ThenFunc(app.userSignupPost))
	// router.Handler(http.MethodGet, "/user/login", dynamic.ThenFunc(app.userLogin))
	// router.Handler(http.MethodPost, "/user/login", dynamic.ThenFunc(app.userLoginPost))

	// protected := dynamic.Append(app.requireAuthentication)
	// router.Handler(http.MethodGet, "/snippet/create", protected.ThenFunc(app.snippetCreate))
	// router.Handler(http.MethodPost, "/snippet/create", protected.ThenFunc(app.snippetCreatePost))
	// router.Handler(http.MethodPost, "/user/logout", protected.ThenFunc(app.userLogoutPost))

	// standard := alice.New(app.recoverPanic, app.logRequest, secureHeaders)

	return router // standard.Then(router)
}
