package main

import (
	"fmt"
	"github.com/hayohtee/books/internal/data"
	"github.com/hayohtee/books/internal/validator"
	"net/http"
	"strings"
	"time"
)

func (app *application) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	var payload RegistrationRequest
	if err := app.readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	validateRegistrationRequest(payload, v)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	passwordHash, err := generatePasswordHash(payload.Password)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	createUserParams := data.CreateUserParams{
		FirstName:    payload.FirstName,
		LastName:     payload.LastName,
		Email:        string(payload.Email),
		PasswordHash: passwordHash,
	}

	row, err := app.queries.CreateUser(r.Context(), createUserParams)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "users_email_key"):
			errResp := Error{Message: "A user with this email already exists."}
			app.errorResponse(w, r, http.StatusConflict, errResp)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	app.background(func() {
		otp, err := generateOTP()
		if err != nil {
			app.logger.Error(fmt.Sprintf("error generating OTP for %s: %v", payload.Email, err))
			return
		}

		templateData := map[string]string{
			"Code": otp,
			"Year": time.Now().String(),
		}

		if err = app.mailer.Send(string(payload.Email), "user_welcome.tmpl", templateData); err != nil {
			app.logger.Error(err.Error())
		}
	})

	resp := UserResponse{
		Id:            row.ID,
		Email:         payload.Email,
		FirstName:     payload.FirstName,
		LastName:      payload.LastName,
		CreatedAt:     row.CreatedAt,
		EmailVerified: row.EmailVerified,
	}

	header := make(http.Header)
	header.Set("Location", fmt.Sprintf("/users/%s", row.ID))

	if err = app.writeJSON(w, http.StatusCreated, resp, header); err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) LoginUserHandler(w http.ResponseWriter, r *http.Request) {

}

func (app *application) ResendCodeHandler(w http.ResponseWriter, r *http.Request) {

}

func (app *application) VerifyEmailHandler(w http.ResponseWriter, r *http.Request) {

}

func (app *application) RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {

}

func validateRegistrationRequest(r RegistrationRequest, v *validator.Validator) {
	v.Check(r.FirstName != "", "first_name", "must be provided")
	v.Check(len(r.FirstName) <= 500, "first_name", "must not be more than 500 bytes long")

	v.Check(r.LastName != "", "last_name", "must be provided")
	v.Check(len(r.LastName) <= 500, "last_name", "must not be more than 500 bytes long")

	v.Check(r.Email != "", "email", "must be provided")
	v.Check(validator.Matches(string(r.Email), validator.EmailRX), "email", "must be a valid email address")

	v.Check(r.Password != "", "password", "must be provided")
	v.Check(len(r.Password) >= 8, "password", "must be at least 8 bytes")
	v.Check(len(r.Password) <= 72, "password", "must not be more than 72 bytes long")
}
