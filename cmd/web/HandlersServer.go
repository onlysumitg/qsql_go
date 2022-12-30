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
func (app *application) ServerListMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		next.ServeHTTP(w, r)
	})

}

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) ServerHandlers(router *chi.Mux) {
	router.Route("/servers", func(r chi.Router) {
		//r.With(paginate).Get("/", listArticles)
		r.Get("/", app.ServerList)
		r.Get("/{serverid}", app.ServerView)

		r.Get("/add", app.ServerAdd)
		r.Post("/add", app.ServerAddPost)

		r.Get("/update/{serverid}", app.ServerUpdate)
		r.Post("/update", app.ServerUpdatePost)

		r.Get("/select/{serverid}", app.ServerSelect)

		r.Get("/delete/{serverid}", app.ServerDelete)
		r.Post("/delete", app.ServerDeleteConfirm)

	})

}

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) ServerList(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Servers = app.servers.List()
	nextUrl := r.URL.Query().Get("next") //filters=["color", "price", "brand"]
	if nextUrl == "" {
		nextUrl = "/query"
	}
	data.Next = nextUrl
	app.render(w, r, http.StatusOK, "server_list.tmpl", data)

}

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) ServerSelect(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "serverid")
	server, err := app.servers.Get(serverID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	if server.OnHold {
		app.sessionManager.Put(r.Context(), "warning", "Server is on hold. Please select a differnt server")

	} else {
		app.sessionManager.Put(r.Context(), "currentserver", server.ID)
		app.sessionManager.Put(r.Context(), "flash", fmt.Sprintf("Selected server: %s", server.Name))

		nextUrl := r.URL.Query().Get("next") //filters=["color", "price", "brand"]

		if nextUrl != "" {
			http.Redirect(w, r, nextUrl, http.StatusSeeOther)
			return
		}
	}

	app.goBack(w, r, http.StatusSeeOther)

	//http.Redirect(w, r, "/query", http.StatusSeeOther)

}

// ------------------------------------------------------
// Server details
// ------------------------------------------------------
func (app *application) ServerView(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	serverID := chi.URLParam(r, "serverid")
	log.Println("serverID >>>", serverID)
	server, err := app.servers.Get(serverID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	data.Server = server
	app.render(w, r, http.StatusOK, "server_view.tmpl", data)

}

// ------------------------------------------------------
// Delete servet
// ------------------------------------------------------
func (app *application) ServerDelete(w http.ResponseWriter, r *http.Request) {

	serverID := chi.URLParam(r, "serverid")

	server, err := app.servers.Get(serverID)
	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("Error deleting server: %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}

	data := app.newTemplateData(r)
	data.Server = server

	app.render(w, r, http.StatusOK, "server_delete.tmpl", data)

}

// ------------------------------------------------------
// Delete servet
// ------------------------------------------------------
func (app *application) ServerDeleteConfirm(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("001 Error processing form %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}

	serverID := r.PostForm.Get("serverid")

	err = app.servers.Delete(serverID)
	if err != nil {

		log.Println("ServerDeleteConfirm  002 >>>>>>", err.Error())
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("Error deleting server: %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}
	app.sessionManager.Put(r.Context(), "flash", "Server deleted sucessfully")

	http.Redirect(w, r, "/servers", http.StatusSeeOther)

}

// ------------------------------------------------------
// add new server
// ------------------------------------------------------
func (app *application) ServerAdd(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	// set form initial values
	data.Form = models.Server{
		Connections: 10,
	}
	app.render(w, r, http.StatusOK, "server_add.tmpl", data)

}

// ----------------------------------------------
func (app *application) ServerAddPost(w http.ResponseWriter, r *http.Request) {
	// Limit the request body size to 4096 bytes
	//r.Body = http.MaxBytesReader(w, r.Body, 4096)

	// r.ParseForm() method to parse the request body. This checks
	// that the request body is well-formed, and then stores the form data in the request’s
	// r.PostForm map.
	err := r.ParseForm()
	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("001 Error processing form %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}

	// Use the r.PostForm.Get() method to retrieve the title and content
	// from the r.PostForm map.
	//	title := r.PostForm.Get("title")
	//	content := r.PostForm.Get("content")

	// the r.PostForm map is populated only for POST , PATCH and PUT requests, and contains the
	// form data from the request body.

	// In contrast, the r.Form map is populated for all requests (irrespective of their HTTP method),

	var server models.Server
	// Call the Decode() method of the form decoder, passing in the current
	// request and *a pointer* to our snippetCreateForm struct. This will
	// essentially fill our struct with the relevant values from the HTML form.
	// If there is a problem, we return a 400 Bad Request response to the client.
	err = app.formDecoder.Decode(&server, r.PostForm)
	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("002 Error processing form %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}
	server.CheckField(!app.servers.DuplicateName(&server), "name", "Duplicate Name")

	server.CheckField(validator.NotBlank(server.Name), "name", "This field cannot be blank")
	server.CheckField(validator.NotBlank(server.IP), "ip", "This field cannot be blank")
	server.CheckField(validator.NotBlank(server.UserName), "user_name", "This field cannot be blank")
	server.CheckField(validator.NotBlank(server.Password), "password", "This field cannot be blank")
	server.CheckField(validator.NotBlank(server.WorkLib), "worklib", "This field cannot be blank")

	// Use the Valid() method to see if any of the checks failed. If they did,
	// then re-render the template passing in the form in the same way as
	// before.

	if !server.Valid() {
		data := app.newTemplateData(r)
		data.Form = server
		app.sessionManager.Put(r.Context(), "error", "Please fix error(s) and resubmit")

		app.render(w, r, http.StatusUnprocessableEntity, "server_add.tmpl", data)
		return
	}

	id, err := app.servers.Insert(&server)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	app.sessionManager.Put(r.Context(), "flash", fmt.Sprintf("Server %s added sucessfully", server.Name))

	http.Redirect(w, r, fmt.Sprintf("/servers/%s", id), http.StatusSeeOther)
}

// ------------------------------------------------------
// add new server
// ------------------------------------------------------
func (app *application) ServerUpdate(w http.ResponseWriter, r *http.Request) {

	serverID := chi.URLParam(r, "serverid")

	server, err := app.servers.Get(serverID)
	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("Error updating server: %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}

	data := app.newTemplateData(r)

	data.Form = server

	app.render(w, r, http.StatusOK, "server_update.tmpl", data)

}

// ----------------------------------------------
func (app *application) ServerUpdatePost(w http.ResponseWriter, r *http.Request) {
	// Limit the request body size to 4096 bytes
	//r.Body = http.MaxBytesReader(w, r.Body, 4096)

	// r.ParseForm() method to parse the request body. This checks
	// that the request body is well-formed, and then stores the form data in the request’s
	// r.PostForm map.
	err := r.ParseForm()
	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("001 Error processing form %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}

	// Use the r.PostForm.Get() method to retrieve the title and content
	// from the r.PostForm map.
	//	title := r.PostForm.Get("title")
	//	content := r.PostForm.Get("content")

	// the r.PostForm map is populated only for POST , PATCH and PUT requests, and contains the
	// form data from the request body.

	// In contrast, the r.Form map is populated for all requests (irrespective of their HTTP method),

	var server models.Server
	// Call the Decode() method of the form decoder, passing in the current
	// request and *a pointer* to our snippetCreateForm struct. This will
	// essentially fill our struct with the relevant values from the HTML form.
	// If there is a problem, we return a 400 Bad Request response to the client.
	err = app.formDecoder.Decode(&server, r.PostForm)

	fmt.Println(">> decord form 1", server)

	if err != nil {
		app.sessionManager.Put(r.Context(), "error", fmt.Sprintf("002 Error processing form %s", err.Error()))
		app.goBack(w, r, http.StatusBadRequest)
		return
	}
	server.CheckField(!app.servers.DuplicateName(&server), "name", "Duplicate Name")

	server.CheckField(validator.NotBlank(server.Name), "name", "This field cannot be blank")
	server.CheckField(validator.NotBlank(server.IP), "ip", "This field cannot be blank")
	server.CheckField(validator.NotBlank(server.UserName), "user_name", "This field cannot be blank")
	server.CheckField(validator.NotBlank(server.Password), "password", "This field cannot be blank")
	server.CheckField(validator.NotBlank(server.WorkLib), "worklib", "This field cannot be blank")

	// Use the Valid() method to see if any of the checks failed. If they did,
	// then re-render the template passing in the form in the same way as
	// before.

	if !server.Valid() {
		data := app.newTemplateData(r)
		data.Form = server
		app.sessionManager.Put(r.Context(), "error", "Please fix error(s) and resubmit")

		app.render(w, r, http.StatusUnprocessableEntity, "server_update.tmpl", data)
		return
	}

	err = app.servers.Update(&server, true)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	app.sessionManager.Put(r.Context(), "flash", fmt.Sprintf("Server %s updated sucessfully", server.Name))

	http.Redirect(w, r, "/servers", http.StatusSeeOther)
}
