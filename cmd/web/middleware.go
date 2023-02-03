package main

import (
	"fmt"
	"net/http"

	"github.com/onlysumitg/qsql2/internal/models"

	"github.com/justinas/nosurf" // New import
)

// Create a NoSurf middleware function which uses a customized CSRF cookie with
// the Secure, Path and HttpOnly attributes set.
func noSurf(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next)
	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
	})
	return csrfHandler
}

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) ClearOldTabId(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, oldTabId := getTabIds(r)

		sessionID := app.sessionManager.Token(r.Context())

		currentSessionId := fmt.Sprintf("%s_%s", sessionID, oldTabId)
		go models.CloseOpenQueries(currentSessionId)

		next.ServeHTTP(w, r)
	})

}
