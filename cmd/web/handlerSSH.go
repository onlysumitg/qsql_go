package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) SSHHandlers(router *chi.Mux) {
	router.Route("/ssh", func(r chi.Router) {
		r.Use(app.CurrentServerMiddleware)
		//r.With(paginate).Get("/", listArticles)
		r.Get("/", app.SSHScreen)

	})

}

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) SSHScreen(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	app.render(w, r, http.StatusOK, "ssh.tmpl", data)

}
