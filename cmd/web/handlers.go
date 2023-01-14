package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/go-chi/chi/v5"
)

func (app *application) langingPage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/servers", http.StatusSeeOther)

}

func (app *application) helpPage(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	app.render(w, r, http.StatusOK, "help.tmpl", data)

}

// ------------------------------------------------------
// basic http operations
// ------------------------------------------------------
func (app *application) helloworld(w http.ResponseWriter, r *http.Request) {
	// headers
	w.WriteHeader(200) // can be used only once --> 2nd call wont have any impact

	w.Header().Set("key", "value")        // can be used multiple times
	w.Header()["key"] = []string{"value"} // type Header map[string][]string

	// http.Error(w,"Method not allowed", 405)
	// http.Error(w,"Method not allowed", http.StatusMethodNotAllowed)  ==> with http constatn

	// http.Error(w,http.StatusText(http.StatusBadRequest), http.StatusBadRequest)

	// http.NotFound(w,r) ==> to return 404
	if r.Method == http.MethodPost {
		w.Write([]byte("POST hello world"))
	}

	// http://localhost:4000/helloworld?id=sumit
	id := r.URL.Query().Get("id")

	messag := "hello world " + id

	w.Write([]byte(messag))

	fmt.Fprintf(w, "\nthis is also id %s", id)

	app.sessionManager.Put(r.Context(), "flash", "HELLOWORLD FLASH MESSAGE")

}

// ------------------------------------------------------
// basis template operations
// ------------------------------------------------------
func templates(w http.ResponseWriter, r *http.Request) {
	files := []string{
		"./ui/html/base.tmpl",
		"./ui/html/pages/home.tmpl",
		"./ui/html/partials/nav.tmpl",
	}

	ts, err := template.ParseFiles(files...)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	err = ts.ExecuteTemplate(w, "base", nil) // render base template ==> defined in base.tmpl => nothing to do with file name
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

}

// ------------------------------------------------------
// advance template operations
// ------------------------------------------------------
func (app *application) templatesAdvance(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	app.render(w, r, http.StatusOK, "view.tmpl", data)

}

// ------------------------------------------------------
// download file
// ------------------------------------------------------
func downloadFileHandler(w http.ResponseWriter, r *http.Request) {
	// need to use filepath.Clean() if download path contains any value based on user input
	http.ServeFile(w, r, "./ui/static/download.zip")

}

// ------------------------------------------------------
// download file
// ------------------------------------------------------
func downloadFileInMemoryHandler(w http.ResponseWriter, r *http.Request) {
	xlsx := excelize.NewFile()
	xlsx.NewSheet("Sheet1")
	xlsx.SetCellValue("Sheet1", "A2", "Hello world.")
	var b bytes.Buffer

	xlsx.Write(&b)

	downloadName := time.Now().UTC().Format("data-20060102150405.xlsx")

	w.Header().Set("Content-Description", "File Transfer")                      // can be used multiple times
	w.Header().Set("Content-Disposition", "attachment; filename="+downloadName) // can be used multiple times
	w.Header().Set("Content-Type", "application/octet-stream")

	w.Write(b.Bytes())
}

// ------------------------------------------------------
// download file
// ------------------------------------------------------
func downloadExcelHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(">>>>>>>>>>>>>>>>>>> downloadexcel >>>>>>>>>>>>>>>>>")
	// need to use filepath.Clean() if download path contains any value based on user input
	id := chi.URLParam(r, "id")
	http.ServeFile(w, r, fmt.Sprintf("./downloads/%s", id))

}
