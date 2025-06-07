package main

import (
	"context"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	middleware "github.com/oapi-codegen/nethttp-middleware"
	"log/slog"
	"net/http"
	"time"
)

func (app *application) serve() error {
	spec, err := GetSwagger()
	if err != nil {
		return err
	}
	fixSwaggerPrefix("/v1", spec)
	mux := http.NewServeMux()
	validator := middleware.OapiRequestValidatorWithOptions(spec, &middleware.Options{
		DoNotValidateServers: true,
		ErrorHandler: func(w http.ResponseWriter, message string, statusCode int) {
			if err := app.writeJSON(w, statusCode, Error{Message: message}, nil); err != nil {
				app.logger.Error(err.Error())
			}
		},
		Options: openapi3filter.Options{
			AuthenticationFunc: func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
				if input.SecuritySchemeName != "BearerAuth" {
					return fmt.Errorf("security scheme %s != 'BearerAuth'", input.SecuritySchemeName)
				}
				return nil
			},
			ExcludeResponseBody: true,
			ExcludeRequestBody:  true,
		},
	})

	srv := http.Server{
		Addr: fmt.Sprintf(":%d", app.cfg.port),
		Handler: HandlerWithOptions(app, StdHTTPServerOptions{
			BaseRouter: mux,
			BaseURL:    "/v1",
			ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
				app.errorResponse(w, r, http.StatusBadRequest, Error{Message: err.Error()})
			},
			Middlewares: []MiddlewareFunc{
				app.recoverPanic,
				app.cors,
				validator,
				app.requireAuthentication,
			},
		}),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
	}

	return srv.ListenAndServe()
}

func fixSwaggerPrefix(prefix string, swagger *openapi3.T) {
	updatedPaths := openapi3.Paths{}

	for key, value := range swagger.Paths.Map() {
		updatedPaths.Set(prefix+key, value)
	}

	swagger.Paths = &updatedPaths
}
