package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// writeJSON is a helper method for sending JSON responses.
//
// It takes the destination http.ResponseWriter, the HTTP status
// code to send, the data to encode to JSON, and a header map
// containing additional HTTP headers to include in the response.
func (app *application) writeJSON(w http.ResponseWriter, status int, data any, header http.Header) error {
	// Encode the data into JSON.
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	// Include the header values in the response
	for key, values := range header {
		w.Header()[key] = values
	}

	// Add the "Content-Type: application/json" header,
	// then write the status code and JSON response.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

// readJSON is a helper method for reading JSON body into the specified
// destination. It also triages all possible errors and return a custom
// helpful error.
func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// Use the http.MaxBytesReader() to limit the size of the request body to 1MB.
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	// Decode the request body to the destination while providing helpful error messages if any.
	if err := dec.Decode(dst); err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)
		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)
		case errors.As(err, &invalidUnmarshalError):
			panic(err)
		default:
			return err
		}
	}

	if err := dec.Decode(&struct{}{}); err != nil {
		return errors.New("body must contain a single JSON value")
	}

	return nil
}
