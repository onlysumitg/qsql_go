package main

import (
	"encoding/json"
	"errors"
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
func (app *application) SavedQueryHandlers(router *chi.Mux) {
	router.Route("/savesql", func(r chi.Router) {
		r.Use(app.CurrentServerMiddleware)
		//r.With(paginate).Get("/", listArticles)
		r.Get("/", app.SavedQueryList)
		r.Get("/{queryid}", app.SavedQueryView)

		r.Get("/add", app.SavedQueryAdd)
		r.Post("/add", app.SavedQueryAddPost)

		r.Get("/update/{queryid}", app.SavedQueryUpdate)

		r.Get("/delete/{queryid}", app.SavedQueryDelete)
		r.Post("/delete", app.SavedQueryDeleteConfirm)

		r.Get("/run/{queryid}", app.SavedQueryRun)
		r.Get("/run", app.SavedQueryRun)

		r.Post("/build", app.SavedQueryBuild)

	})

}

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) SavedQueryList(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.SavesQueries = app.savedQueries.List()
	nextUrl := r.URL.Query().Get("next") //filters=["color", "price", "brand"]
	data.Next = nextUrl
	app.render(w, r, http.StatusOK, "query_saved_list.tmpl", data)

}

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) SavedQueryRun(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	savesQueries := app.savedQueries.List()
	data.SavesQueries = savesQueries
	data.SavesQueriesByCategory = make(map[string][]*models.SavedQuery)

	//queryID := chi.URLParam(r, "queryid")

	for _, savesQuery := range savesQueries {
		savesQuery.PopulateFields()

		queryList, found := data.SavesQueriesByCategory[savesQuery.Category]
		if !found {
			queryList = make([]*models.SavedQuery, 0)
		}
		queryList = append(queryList, savesQuery)
		data.SavesQueriesByCategory[savesQuery.Category] = queryList

	}

	nextUrl := r.URL.Query().Get("next") //filters=["color", "price", "brand"]
	data.Next = nextUrl
	app.render(w, r, http.StatusOK, "query_saved_run.tmpl", data)

}

// ------------------------------------------------------
func (app *application) SavedQueryBuild(w http.ResponseWriter, r *http.Request) {

	formMap := map[string]string{}
	err := json.NewDecoder(r.Body).Decode(&formMap)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}
	log.Println("><<>>>>>>", formMap)
	savedQueeryId, found := formMap["savedqueryid"]
	if savedQueeryId == "" || !found {
		app.serverError(w, r, errors.New("savedqueryid is required"))
		return
	}
	savedQuery, err := app.savedQueries.Get(savedQueeryId)
	log.Println("savedQuery>>>", savedQuery, err)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	sqlToRun, fieldError := savedQuery.ReplaceFields(formMap)
	if len(fieldError) > 0 {
		// if has error field -> return blank sql to run
		sqlToRun = ""

	}

	savedQueryBuild := models.SavedQueryBuild{SqlToRun: sqlToRun, FieldErrors: fieldError}

	app.writeJSON(w, http.StatusOK, savedQueryBuild, nil)

	// need to return a json

}

// ------------------------------------------------------
func (app *application) SavedQueryRunAsJson(w http.ResponseWriter, r *http.Request) {

	currentServerID := app.sessionManager.GetString(r.Context(), "currentserver")
	currentServer, err := app.servers.Get(currentServerID)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if err := r.ParseForm(); err != nil {
		// handle error
	}
	savedQueeryId := r.PostForm.Get("savedqueryid")
	if savedQueeryId != "" {
		app.serverError(w, r, errors.New("savedqueryid is required"))
		return
	}
	savedQuery, err := app.savedQueries.Get(savedQueeryId)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	fieldMap := make(map[string]string)
	for key, values := range r.PostForm {
		fieldMap[key] = values[0]
	}

	sqlToRun, fieldError := savedQuery.ReplaceFields(fieldMap)
	if len(fieldError) == 0 {
		// No error
		// run the sql
	}

	queryResults := models.ProcessSQLStatements(sqlToRun, currentServer)
	app.writeJSON(w, http.StatusOK, queryResults, nil)

}

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) SavedQueryAdd(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = models.SavedQuery{}
	app.render(w, r, http.StatusOK, "query_saved_add.tmpl", data)

}

// ------------------------------------------------------
// Server details
// ------------------------------------------------------
func (app *application) SavedQueryView(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	queryID := chi.URLParam(r, "queryid")
	log.Println("queryid >>>", queryID)
	savedQuery, err := app.savedQueries.Get(queryID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	data.SavesQuery = savedQuery
	app.render(w, r, http.StatusOK, "query_saved_view.tmpl", data)

}

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) SavedQueryAddPost(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("001 Error processing form %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}

	var savedQuery models.SavedQuery
	err = app.formDecoder.Decode(&savedQuery, r.PostForm)
	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("002 Error processing form %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}

	savedQuery.CheckField(validator.NotBlank(savedQuery.Name), "name", "This field cannot be blank")
	savedQuery.CheckField(validator.NotBlank(savedQuery.Category), "category", "This field cannot be blank")
	savedQuery.CheckField(validator.NotBlank(savedQuery.Sql), "sql", "This field cannot be blank")
	if !savedQuery.Valid() {
		data := app.newTemplateData(r)
		data.Form = savedQuery
		app.sessionManager.Put(r.Context(), "error", "Please fix error(s) and resubmit")

		app.render(w, r, http.StatusUnprocessableEntity, "query_saved_add.tmpl", data)
		return
	}

	id, err := app.savedQueries.Save(&savedQuery)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	fmt.Println("id", id)
	app.sessionManager.Put(r.Context(), "flash", fmt.Sprintf("Query %s added sucessfully", savedQuery.Name))

	//http.Redirect(w, r, fmt.Sprintf("/savesql/%s", id), http.StatusSeeOther)
	http.Redirect(w, r, "/savesql/add", http.StatusSeeOther)

}

// ------------------------------------------------------
// Delete saved query
// ------------------------------------------------------
func (app *application) SavedQueryDelete(w http.ResponseWriter, r *http.Request) {

	queryid := chi.URLParam(r, "queryid")

	savedQuery, err := app.savedQueries.Get(queryid)
	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("Error deleting query: %s", err.Error()))
		app.goBack(w, r, http.StatusSeeOther)
		return
	}

	data := app.newTemplateData(r)
	data.SavesQuery = savedQuery

	app.render(w, r, http.StatusOK, "query_saved_delete.tmpl", data)

}

// ------------------------------------------------------
// Delete saved query confirm
// ------------------------------------------------------
func (app *application) SavedQueryDeleteConfirm(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("001 Error processing form %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}

	queryid := r.PostForm.Get("queryid")

	err = app.savedQueries.Delete(queryid)
	if err != nil {

		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("Error deleting query: %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}
	app.sessionManager.Put(r.Context(), "flash", "Query deleted sucessfully")

	http.Redirect(w, r, "/savesql", http.StatusSeeOther)

}

// ------------------------------------------------------
// add new server
// ------------------------------------------------------
func (app *application) SavedQueryUpdate(w http.ResponseWriter, r *http.Request) {

	queryid := chi.URLParam(r, "queryid")

	savedQuery, err := app.savedQueries.Get(queryid)
	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("Error updating server: %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}

	data := app.newTemplateData(r)

	data.Form = savedQuery

	app.render(w, r, http.StatusOK, "query_saved_add.tmpl", data)

}
