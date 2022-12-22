package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) BatchQueryHandlers(router *chi.Mux) {
	router.Route("/batchquery", func(r chi.Router) {
		//r.With(paginate).Get("/", listArticles)
		r.Get("/", app.BatchQueryList)
		r.Get("/{queryid}", app.BatchQueryView)

	})

}

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) BatchQueryList(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.BatchQueries = app.batchSQLModel.List()
	nextUrl := r.URL.Query().Get("next") //filters=["color", "price", "brand"]
	data.Next = nextUrl
	app.render(w, r, http.StatusOK, "batch_query_list.tmpl", data)

}

func (app *application) BatchQueryView(w http.ResponseWriter, r *http.Request) {
	queryid := chi.URLParam(r, "queryid")
	data := app.newTemplateData(r)

	batchQuery, err := app.batchSQLModel.Get(queryid)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	data.BatchQuery = batchQuery
	app.render(w, r, http.StatusOK, "batch_query_result.tmpl", data)
}
