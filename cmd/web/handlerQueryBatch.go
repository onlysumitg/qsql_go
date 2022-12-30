package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/onlysumitg/qsql2/internal/models"
	"github.com/onlysumitg/qsql2/internal/validator"
)

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) BatchQueryHandlers(router *chi.Mux) {
	router.Route("/batchquery", func(r chi.Router) {
		//r.With(paginate).Get("/", listArticles)
		r.Get("/", app.BatchQueryList)
		//r.Get("/{queryid}", app.BatchQueryView)

		r.Get("/add", app.BatchQueryAdd)
		r.Post("/add", app.BatchQueryAddPost)

		r.Get("/delete/{queryid}", app.BatchQueryDelete)
		r.Post("/delete", app.BatchQueryDeleteConfirm)

		r.Get("/runs/{queryid}", app.BatchQueryRunList)
		r.Get("/runs/result/{runid}", app.BatchQueryRun)

	})

}

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) BatchQueryAdd(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = models.BatchSqlForm{}
	app.render(w, r, http.StatusOK, "batch_query_add.tmpl", data)

}

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) BatchQueryAddPost(w http.ResponseWriter, r *http.Request) {
	currentServerID := app.sessionManager.GetString(r.Context(), "currentserver")
	currentServer, err := app.servers.Get(currentServerID)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}

	err = r.ParseForm()
	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("001 Error processing form %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}

	var batchSqlForm models.BatchSqlForm
	err = app.formDecoder.Decode(&batchSqlForm, r.PostForm)
	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("002 Error processing form %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}

	batchSqlForm.CheckField(validator.NotBlank(batchSqlForm.Sql), "sql", "This field cannot be blank")
	batchSqlForm.CheckField(!validator.MustStartwith(batchSqlForm.Sql, "@BATCH"), "sql", "Query can not start with a @BATCH prefix.")

	// should not submit multiple queries

	if !batchSqlForm.Valid() {
		data := app.newTemplateData(r)
		data.Form = batchSqlForm
		app.sessionManager.Put(r.Context(), "error", "Please fix error(s) and resubmit")

		app.render(w, r, http.StatusUnprocessableEntity, "batch_query_add.tmpl", data)
		return
	}

	//RunningSql
	currentSql := &models.RunningSql{}
	currentSql.ID = uuid.NewString()
	currentSql.Sql = batchSqlForm.Sql

	batchSql := models.BatchSql{Server: *currentServer,
		RunningSql:   *currentSql,
		RepeatEvery:  time.Duration(batchSqlForm.RepeatEvery) * time.Minute,
		RepeatXtimes: batchSqlForm.RepeatXtimes,
	}

	_, err = app.batchSQLModel.Insert(&batchSql)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	app.sessionManager.Put(r.Context(), "flash", "Query added sucessfully")

	http.Redirect(w, r, "/batchquery", http.StatusSeeOther)
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

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) BatchQueryRunList(w http.ResponseWriter, r *http.Request) {
	queryid := chi.URLParam(r, "queryid")
	data := app.newTemplateData(r)

	batchQuery, err := app.batchSQLModel.Get(queryid)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data.BatchQuertRuns = app.batchSQLModel.ListRun(batchQuery)
	data.BatchQuery = batchQuery

	if batchQuery.RepeatXtimes == 1 && len(data.BatchQuertRuns) == 1 {
		run := data.BatchQuertRuns[0]
		http.Redirect(w, r, fmt.Sprintf("/batchquery/runs/result/%s", run.ID), http.StatusSeeOther)

	}

	// if only one run --> redirect to /runs/result/{runid}

	app.render(w, r, http.StatusOK, "batch_query_run_list.tmpl", data)
}

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) BatchQueryRun(w http.ResponseWriter, r *http.Request) {
	runid := chi.URLParam(r, "runid")
	data := app.newTemplateData(r)

	batchQueryRun, err := app.batchSQLModel.GetRun(runid)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	data.BatchQueryRun = batchQueryRun

	batchQuery, err := app.batchSQLModel.Get(batchQueryRun.ParentId)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	data.BatchQuery = batchQuery

	app.render(w, r, http.StatusOK, "batch_query_run_result.tmpl", data)
}

// ------------------------------------------------------
//
// ------------------------------------------------------
// func (app *application) BatchQueryView(w http.ResponseWriter, r *http.Request) {
// 	queryid := chi.URLParam(r, "queryid")
// 	data := app.newTemplateData(r)

// 	batchQuery, err := app.batchSQLModel.Get(queryid)
// 	if err != nil {
// 		app.serverError(w, r, err)
// 		return
// 	}
// 	data.BatchQuery = batchQuery
// 	app.render(w, r, http.StatusOK, "batch_query_result.tmpl", data)
// }

// ------------------------------------------------------
// Delete servet
// ------------------------------------------------------
func (app *application) BatchQueryDelete(w http.ResponseWriter, r *http.Request) {

	queryid := chi.URLParam(r, "queryid")

	query, err := app.batchSQLModel.Get(queryid)
	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("Error deleting server: %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}

	data := app.newTemplateData(r)
	data.BatchQuery = query

	app.render(w, r, http.StatusOK, "batch_query_delete.tmpl", data)

}

// ------------------------------------------------------
// Delete servet
// ------------------------------------------------------
func (app *application) BatchQueryDeleteConfirm(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("001 Error processing form %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}

	queryid := r.PostForm.Get("queryid")

	err = app.batchSQLModel.Delete(queryid)
	if err != nil {

		log.Println("ServerDeleteConfirm  002 >>>>>>", err.Error())
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("Error deleting server: %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}
	app.sessionManager.Put(r.Context(), "flash", "Deleted sucessfully")

	http.Redirect(w, r, "/batchquery", http.StatusSeeOther)

}
