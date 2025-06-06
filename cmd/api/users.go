package main

import (
	"database/sql"
	"errors"
	openapitypes "github.com/oapi-codegen/runtime/types"
	"net/http"
)

func (app *application) GetUserHandler(w http.ResponseWriter, r *http.Request, id openapitypes.UUID) {
	user, err := app.queries.GetUser(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			app.errorResponse(w, r, http.StatusNotFound, Error{Message: "user not found"})
		default:
			app.serverError(w, r, err)
		}
		return
	}

	resp := UserResponse{
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		CreatedAt:     user.CreatedAt,
		Id:            user.ID,
		Email:         openapitypes.Email(user.Email),
		EmailVerified: user.EmailVerified,
	}

	if err = app.writeJSON(w, http.StatusOK, resp, nil); err != nil {
		app.serverError(w, r, err)
	}
}
