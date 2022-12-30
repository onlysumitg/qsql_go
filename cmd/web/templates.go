package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"time"

	"github.com/justinas/nosurf"
	"github.com/onlysumitg/qsql2/internal/models"
	"github.com/onlysumitg/qsql2/ui"
)

type templateData struct {
	CurrentYear int

	HostUrl string

	Form any //use this Form field to pass the validation errors and previously submitted data back to the template when we re-display the form.

	// differnt notifications
	Flash   string
	Warning string
	Error   string

	IsAuthenticated bool

	CSRFToken string // Add a CSRFToken field.   <input type='hidden' name='csrf_token' value='{{.CSRFToken}}'>

	Server        *models.Server
	Servers       []*models.Server
	CurrentServer *models.Server

	QueryResults []models.QueryResult

	SavesQueries []*models.SavedQuery
	SavesQuery   *models.SavedQuery

	SavesQueriesByCategory map[string][]*models.SavedQuery

	BatchQuery   *models.BatchSql
	BatchQueries []*models.BatchSql

	BatchQueryRun  *models.BatchSQLRun
	BatchQuertRuns []*models.BatchSQLRun

	RepeatQuery   *models.BatchSql
	RepeatQueries []*models.BatchSql
	// ProcessingSavedQuery *models.SavedQuery
	// SavedQueryFields     []*models.QueryField

	ShorthandQueries []*models.ShorthandQuery
	ShortHandQuery   *models.ShorthandQuery

	Next string
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (app *application) newTemplateData(r *http.Request) *templateData {
	return &templateData{
		CurrentYear: time.Now().Year(),

		CSRFToken: nosurf.Token(r), // Add the CSRF token.

		HostUrl: app.hostURL,
	}
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (app *application) setTemplateDefaults(r *http.Request, templateData *templateData) {

	// Add the flash message to the template data, if one exists.
	templateData.Flash = app.sessionManager.PopString(r.Context(), "flash")
	templateData.Warning = app.sessionManager.PopString(r.Context(), "warning")
	templateData.Error = app.sessionManager.PopString(r.Context(), "error")
	currentServerID := app.sessionManager.GetString(r.Context(), "currentserver")

	currentServer, err := app.servers.Get(currentServerID)
	if err == nil {
		templateData.CurrentServer = currentServer
	}

	if templateData.Servers == nil {
		templateData.Servers = app.servers.List()
	}
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (app *application) render(w http.ResponseWriter, r *http.Request, status int, page string, data *templateData) {
	ts, ok := app.templateCache[page]
	if !ok {
		err := fmt.Errorf("the template %s does not exist", page)
		app.serverError(w, r, err)
		return
	}
	// Initialize a new buffer.
	buf := new(bytes.Buffer)

	app.setTemplateDefaults(r, data)

	// Write the template to the buffer, instead of straight to the
	// http.ResponseWriter. If there's an error, call our serverError() helper
	// and then return.
	err := ts.ExecuteTemplate(buf, "base", data)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	// If the template is written to the buffer without any errors, we are safe
	// to go ahead and write the HTTP status code to http.ResponseWriter.
	w.WriteHeader(status)
	// Write the contents of the buffer to the http.ResponseWriter. Note: this
	// is another time where we pass our http.ResponseWriter to a function that
	// takes an io.Writer.
	buf.WriteTo(w)
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------

func toJson(s interface{}) string {
	jsonBytes, err := json.Marshal(s)
	if err != nil {
		return ""
	}

	return string(jsonBytes)
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func yesNo(s bool) string {
	if s {
		return "Yes"
	}

	return "No"
}

func humanDate(t time.Time) string {
	// Return the empty string if time has the zero value.
	if t.IsZero() {
		return ""
	}
	// Convert the time to UTC before formatting it.
	//time.Kitchen
	return t.Local().Format("02 Jan 2006 at 03:04:05PM")
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// Initialize a template.FuncMap object and store it in a global variable. This is essentially
// a string-keyed map which acts as a lookup between the names of our custom template
// functions and the functions themselves.
var functions = template.FuncMap{
	"humanDate": humanDate,
	"toJson":    toJson,
	"yesNo":     yesNo,
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}
	// Use fs.Glob() to get a slice of all filepaths in the ui.Files embedded
	// filesystem which match the pattern 'html/pages/*.tmpl'. This essentially
	// gives us a slice of all the 'page' templates for the application, just
	// like before.
	pages, err := fs.Glob(ui.Files, "html/pages/*.tmpl")
	if err != nil {
		return nil, err
	}
	for _, page := range pages {
		name := filepath.Base(page)
		// Create a slice containing the filepath patterns for the templates we
		// want to parse.
		patterns := []string{
			"html/base.tmpl",
			"html/partials/*.tmpl",
			page,
		}
		// Use ParseFS() instead of ParseFiles() to parse the template files
		// from the ui.Files embedded filesystem.
		ts, err := template.New(name).Funcs(functions).ParseFS(ui.Files, patterns...)
		if err != nil {
			return nil, err
		}
		cache[name] = ts
	}
	return cache, nil
}
