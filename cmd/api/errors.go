package main

import (
	"log/slog"
	"net/http"
)

// logError is a generic helper for logging an error message along with the current request method and URL
// as attributes in the log entry.
func (app *application) logError(r *http.Request, err error) {
	app.logger.Error(err.Error(), slog.String("method", r.Method), slog.String("uri", r.URL.RequestURI()))
}

// errorResponse is a generic method for sending JSON-formatted error messages to the client with the given status code.
func (app *application) errorResponse(w http.ResponseWriter, r *http.Request, status int, data any) {
	if err := app.writeJSON(w, status, data, nil); err != nil {
		app.logError(r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// serverErrorResponse is a helper method for sending JSON-formatted error response
// when the server encountered an unexpected problem at runtime.
//
// It logs the detailed error message, and sends a 500 Internal Server Error
// status code and JSON response containing generic error message to the client.
func (app *application) serverError(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)
	errResp := Error{Message: "the server encountered a problem and could not process the request"}
	app.errorResponse(w, r, http.StatusInternalServerError, errResp)
}

// failedValidationResponse is a helper method for sending a 422 Unprocessable Entity
// status code and JSON response containing the errors.
func (app *application) failedValidationResponse(w http.ResponseWriter, r *http.Request, errs map[string]string) {
	errResp := ValidationError{Message: "the provided input failed validation check"}
	for key, value := range errs {
		fieldError := FieldError{Field: key, Message: value}
		errResp.Errors = append(errResp.Errors, fieldError)
	}

	app.errorResponse(w, r, http.StatusUnprocessableEntity, errResp)
}

// editConflictResponse is a helper method for sending 409 Conflict status code
// and JSON response to the client
func (app *application) editConflictResponse(w http.ResponseWriter, r *http.Request) {
	errResp := Error{Message: "unable to update the record due to an edit conflict, please try again"}
	app.errorResponse(w, r, http.StatusConflict, errResp)
}

// notFoundResponse is a helper method for sending a 404 Not Found status code
// and JSON response to the client.
func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	errResp := Error{Message: "the requested resource could not be found"}
	app.errorResponse(w, r, http.StatusNotFound, errResp)
}

// badRequestResponse is a helper method for sending 400 Bad Request status code
// and the error message as JSON response to the client.
func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	errResp := Error{Message: err.Error()}
	app.errorResponse(w, r, http.StatusBadRequest, errResp)
}

func (app *application) invalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	errResp := Error{Message: "invalid authentication credentials"}
	app.errorResponse(w, r, http.StatusUnauthorized, errResp)
}

func (app *application) emailAddressNotFoundResponse(w http.ResponseWriter, r *http.Request) {
	errResp := Error{Message: "no account exist for the provided email address"}
	app.errorResponse(w, r, http.StatusNotFound, errResp)
}

func (app *application) invalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", "Bearer")

	errResp := Error{Message: "invalid or missing authentication token"}
	app.errorResponse(w, r, http.StatusUnauthorized, errResp)
}

// invalidRefreshTokenResponse is a helper method for sending a JSON response when
// the provided refresh token is either invalid or has expired. It sends an appropriate
// error message to the client.
func (app *application) invalidRefreshTokenResponse(w http.ResponseWriter, r *http.Request) {
	errResp := Error{Message: "the refresh token provided is invalid or expired"}
	app.errorResponse(w, r, http.StatusUnauthorized, errResp)
}
