package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/go-playground/form/v4"
)

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// Create a new decodePostForm() helper method. The second parameter here, dst,
// is the target destination that we want to decode the form data into.
func (app *application) decodePostForm(r *http.Request, dst any) error {
	// Call ParseForm() on the request, in the same way that we did in our
	// createSnippetPost handler.
	err := r.ParseForm()
	if err != nil {
		return err
	}
	// Call Decode() on our decoder instance, passing the target destination as
	// the first parameter.
	err = app.formDecoder.Decode(dst, r.PostForm)
	if err != nil {
		// If we try to use an invalid target destination, the Decode() method
		// will return an error with the type *form.InvalidDecoderError.We use
		// errors.As() to check for this and raise a panic rather than returning
		// the error.
		var invalidDecoderError *form.InvalidDecoderError
		if errors.As(err, &invalidDecoderError) {
			panic(err)
		}
		// For all other errors, we return them as normal.
		return err
	}
	return nil
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
type envelope map[string]interface{}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// Define a writeJSON() helper for sending responses. This takes the destination
// http.ResponseWriter, the HTTP status code to send, the data to encode to JSON, and a
// header map containing any additional HTTP headers we want to include in the response.
func (app *application) writeJSON(w http.ResponseWriter, status int, data interface{}, headers http.Header) error {
	// Encode the data to JSON, returning the error if there was one.
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}
	// Append a newline to make it easier to view in terminal applications.
	js = append(js, '\n')
	// At this point, we know that we won't encounter any more errors before writing the
	// response, so it's safe to add any headers that we want to include. We loop
	// through the header map and add each header to the http.ResponseWriter header map.
	// Note that it's OK if the provided header map is nil. Go doesn't throw an error
	// if you try to range over (or generally, read from) a nil map.
	for key, value := range headers {
		w.Header()[key] = value
	}
	// Add the "Content-Type: application/json" header, then write the status code and
	// JSON response.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)
	return nil
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (app *application) goBack(w http.ResponseWriter, r *http.Request, status int) {

	log.Println("r.Header.Get(Referer) >>>>>>>>>", r.Header.Get("Referer"))
	http.Redirect(w, r, r.Header.Get("Referer"), status)
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (app *application) serverError(w http.ResponseWriter, r *http.Request, err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	// app.errorLog.Println(trace)
	app.errorLog.Output(2, trace) // make sure error log does not show this helper.go as the error trigger
	// http.Error(w, err.Error(), http.StatusInternalServerError)

	data := app.newTemplateData(r)

	app.sessionManager.Put(r.Context(), "error", err.Error())

	app.render(w, r, http.StatusUnprocessableEntity, "500.tmpl", data)

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (app *application) clientError(w http.ResponseWriter, status int, err error) {
	message := http.StatusText(status)
	if err != nil {
		message = err.Error()
	}
	http.Error(w, message, status)

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (app *application) notFound(w http.ResponseWriter, err error) {
	app.clientError(w, http.StatusNotFound, err)
}

// The errorResponse() method is a generic helper for sending JSON-formatted error
// messages to the client with a given status code. Note that we're using an interface{}
// type for the message parameter, rather than just a string type, as this gives us
// more flexibility over the values that we can include in the response.
func (app *application) errorResponse(w http.ResponseWriter, r *http.Request, status int, message interface{}) {
	env := envelope{"error": message}
	// Write the response using the writeJSON() helper. If this happens to return an
	// error then log it, and fall back to sending the client an empty response with a
	// 500 Internal Server Error status code.
	err := app.writeJSON(w, status, env, nil)
	if err != nil {
		//app.logError(r, err)
		w.WriteHeader(500)
	}
}
