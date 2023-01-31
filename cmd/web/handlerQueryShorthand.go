package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/onlysumitg/qsql2/internal/models"
	"github.com/onlysumitg/qsql2/internal/validator"
)

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) ShorthandQueryHandlers(router *chi.Mux) {
	router.Route("/queryalias", func(r chi.Router) {
		r.Use(app.CurrentServerMiddleware)
		//r.With(paginate).Get("/", listArticles)
		r.Get("/", app.ShorthandQueryList)
		r.Get("/{queryid}", app.ShorthandQueryView)

		r.Get("/add", app.ShorthandQueryAdd)
		r.Post("/add", app.ShorthandQueryAddPost)

		r.Get("/update/{queryid}", app.ShorthandQueryUpdate)

		r.Get("/delete/{queryid}", app.ShorthandQueryDelete)
		r.Post("/delete", app.ShorthandQueryDeleteConfirm)

	})

}

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) ShorthandQueryList(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.ShorthandQueries = app.shorthandQueries.List()
	nextUrl := r.URL.Query().Get("next") //filters=["color", "price", "brand"]
	data.Next = nextUrl
	app.render(w, r, http.StatusOK, "query_alias_list.tmpl", data)

}

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) ShorthandQueryAdd(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = models.ShorthandQuery{}
	app.render(w, r, http.StatusOK, "query_alias_add.tmpl", data)

}

// ------------------------------------------------------
// Server details
// ------------------------------------------------------
func (app *application) ShorthandQueryView(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	queryID := chi.URLParam(r, "queryid")
	log.Println("queryid >>>", queryID)
	shorthandQuery, err := app.shorthandQueries.Get(queryID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	data.ShortHandQuery = shorthandQuery
	app.render(w, r, http.StatusOK, "query_alias_view.tmpl", data)

}

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) ShorthandQueryAddPost(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("001 Error processing form %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}

	var shorthandQuery models.ShorthandQuery
	err = app.formDecoder.Decode(&shorthandQuery, r.PostForm)
	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("002 Error processing form %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}

	shorthandQuery.CheckField(!app.shorthandQueries.DuplicateName(&shorthandQuery), "name", "Duplicate Name")

	shorthandQuery.CheckField(validator.MustStartwith(shorthandQuery.Name, "@"), "name", "Must start with @")
	shorthandQuery.CheckField(validator.CanNotBe(shorthandQuery.Name, "@Batch"), "name", "@Batch is not allowed")
	shorthandQuery.CheckField(validator.CanNotBe(shorthandQuery.Name, "@heading"), "name", "@Heading is not allowed")

	shorthandQuery.CheckField(validator.MustNotContainBlanks(shorthandQuery.Name), "name", "Spaces are not allowed")

	shorthandQuery.CheckField(validator.NotBlank(shorthandQuery.Name), "name", "This field cannot be blank")
	shorthandQuery.CheckField(validator.NotBlank(shorthandQuery.Sql), "sql", "This field cannot be blank")
	if !shorthandQuery.Valid() {
		data := app.newTemplateData(r)
		data.Form = shorthandQuery
		app.sessionManager.Put(r.Context(), "error", "Please fix error(s) and resubmit")

		app.render(w, r, http.StatusUnprocessableEntity, "query_alias_add.tmpl", data)
		return
	}

	id, err := app.shorthandQueries.Save(&shorthandQuery)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	app.sessionManager.Put(r.Context(), "flash", fmt.Sprintf("Query %s saved sucessfully", shorthandQuery.Name))

	go models.LoadQueryMap(app.shorthandQueries, app.savedQueries)

	http.Redirect(w, r, fmt.Sprintf("/queryalias/%s", id), http.StatusSeeOther)
}

// ------------------------------------------------------
// Delete saved query
// ------------------------------------------------------
func (app *application) ShorthandQueryDelete(w http.ResponseWriter, r *http.Request) {

	queryid := chi.URLParam(r, "queryid")

	shorthandQuery, err := app.shorthandQueries.Get(queryid)
	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("Error deleting query: %s", err.Error()))
		app.goBack(w, r, http.StatusSeeOther)
		return
	}

	data := app.newTemplateData(r)
	data.ShortHandQuery = shorthandQuery

	app.render(w, r, http.StatusOK, "query_alias_delete.tmpl", data)

}

// ------------------------------------------------------
// Delete saved query confirm
// ------------------------------------------------------
func (app *application) ShorthandQueryDeleteConfirm(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("001 Error processing form %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}

	queryid := r.PostForm.Get("queryid")

	err = app.shorthandQueries.Delete(queryid)
	if err != nil {

		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("Error deleting query: %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}
	app.sessionManager.Put(r.Context(), "flash", "Query deleted sucessfully")
	go models.LoadQueryMap(app.shorthandQueries, app.savedQueries)

	http.Redirect(w, r, "/queryalias", http.StatusSeeOther)

}

// ------------------------------------------------------
// add new server
// ------------------------------------------------------
func (app *application) ShorthandQueryUpdate(w http.ResponseWriter, r *http.Request) {

	queryid := chi.URLParam(r, "queryid")

	shorthandQuery, err := app.shorthandQueries.Get(queryid)
	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("Error updating server: %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}

	data := app.newTemplateData(r)

	data.Form = shorthandQuery

	app.render(w, r, http.StatusOK, "query_alias_add.tmpl", data)

}
