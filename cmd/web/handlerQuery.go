package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/onlysumitg/qsql2/internal/models"
)

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) CurrentServerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		currentServerID := app.sessionManager.GetString(r.Context(), "currentserver")
		server, err2 := app.servers.Get(currentServerID)

		continueToNext := true
		message := "Please select a server"
		if err2 != nil {
			continueToNext = false
		} else if server.OnHold {
			continueToNext = false
			message = "Server is on hold. Please select a differnt server"
			app.sessionManager.Remove(r.Context(), "currentserver")

		}

		if continueToNext {
			next.ServeHTTP(w, r)
		} else {
			app.sessionManager.Put(r.Context(), "warning", message)

			goToUrl := fmt.Sprintf("/servers?next=%s", r.URL.RequestURI())

			reponseMap := make(map[string]string)
			reponseMap["redirectTo"] = goToUrl
			switch r.Header.Get("Accept") {

			case "application/json":
				app.writeJSON(w, http.StatusSeeOther, reponseMap, nil)

			default:
				http.Redirect(w, r, goToUrl, http.StatusSeeOther)
			}
		}

	})

}

// ------------------------------------------------------
//
// ------------------------------------------------------
type RunQueryReqest struct {
	SQLToRun string
}

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) QueryHandlers(router *chi.Mux) {
	router.Route("/query", func(r chi.Router) {
		r.Use(app.CurrentServerMiddleware)
		//r.With(paginate).Get("/", listArticles)
		r.Get("/", app.QueryScreen)
		r.Post("/run", app.RunQueryPostAsync)
		r.Post("/loadmore", app.LoadMoreQueryPost)

	})

}

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) QueryScreen(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	app.render(w, r, http.StatusOK, "query.tmpl", data)

}

// ------------------------------------------------------
//
// ------------------------------------------------------

func (app *application) LoadMoreQueryPost(w http.ResponseWriter, r *http.Request) {
	currentServerID := app.sessionManager.GetString(r.Context(), "currentserver")
	currentServer, err := app.servers.Get(currentServerID)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}

	runningSQL := &models.RunningSql{}
	err = json.NewDecoder(r.Body).Decode(&runningSQL)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}

	models.PrepareSQLToRun(runningSQL)
	queryResult := models.ActuallyRunSQL2(currentServer, *runningSQL)

	go app.servers.Update(currentServer, false)

	app.writeJSON(w, http.StatusOK, queryResult[0], nil)

}

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) RunQueryPostAsync(w http.ResponseWriter, r *http.Request) {

	log.Println("RunQueryPostAsync>>>>> >>>>>>")

	currentServerID := app.sessionManager.GetString(r.Context(), "currentserver")
	currentServer, err := app.servers.Get(currentServerID)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}

	request := &RunQueryReqest{}

	// Initialize a new json.Decoder instance which reads from the request body, and
	// then use the Decode() method to decode the body contents into the input struct.
	// Importantly, notice that when we call Decode() we pass a *pointer* to the input
	// struct as the target decode destination. If there was an error during decoding,
	// we also use our generic errorResponse() helper to send the client a 400 Bad
	// Request response containing the error message.
	err = json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}

	queryResults := models.ProcessSQLStatements(request.SQLToRun, currentServer)

	for _, queryResult := range queryResults {

		switch {
		case queryResult.CurrentSql.StatementType == "@BATCH":
			// create batch record
			batchSql := &models.BatchSql{Server: *currentServer,
				RunningSql:   queryResult.CurrentSql,
				RepeatXtimes: 1,
				RepeatEvery:  time.Second * 15,
			}

			app.batchSQLModel.Insert(batchSql)

			// case queryResult.CurrentSql.StatementType == "@DOWNLOAD":
			// 	fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>.  DOWNLOASINF >>>>>>>>>>>>>>>>>>>>>>>>")

			// 	downloadName := time.Now().UTC().Format("data-20060102150405.xlsx")
			// 	w.Header().Set("Content-Description", "File Transfer")                      // can be used multiple times
			// 	w.Header().Set("Content-Disposition", "attachment; filename="+downloadName) // can be used multiple times
			// 	w.Header().Set("Content-Type", "application/octet-stream")

			// 	w.Write(queryResult.ToExcel())

			// 	return
		}

	}

	go app.servers.Update(currentServer, false)

	// w.Header().Set("Content-Type", "application/json")
	// w.Write(queryResultsJson)
	app.writeJSON(w, http.StatusOK, queryResults, nil)
}
