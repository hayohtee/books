package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/hayohtee/books/internal/cache"
	"github.com/hayohtee/books/internal/data"
	"github.com/hayohtee/books/internal/validator"
	"net/http"
	"strings"
	"time"
)

const (
	accessTokenDuration  = 2 * time.Hour
	refreshTokenDuration = 24 * time.Hour * 7
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

		for range 5 {
			if err = app.mailer.Send(string(payload.Email), "user_welcome.tmpl", templateData); err != nil {
				app.logger.Error(err.Error())
			} else {
				app.logger.Info("Successfully sent Welcome email to %s", string(payload.Email))
				break
			}
			time.Sleep(5 * time.Second)
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
	var payload LoginRequest
	if err := app.readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	validateLoginRequest(payload, v)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.queries.FindUserByEmail(r.Context(), string(payload.Email))
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			app.errorResponse(w, r, http.StatusUnauthorized, Error{Message: "No account with this email exists."})
		default:
			app.serverError(w, r, err)
		}
		return
	}

	matches, err := passwordMatches(payload.Password, user.PasswordHash)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	if !matches {
		app.invalidCredentialsResponse(w, r)
		return
	}

	accessToken, err := app.cache.NewToken(user.ID, accessTokenDuration, cache.AccessTokenScope)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	refreshToken, err := app.cache.NewToken(user.ID, refreshTokenDuration, cache.RefreshTokenScope)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	resp := TokenResponse{
		AccessToken:  accessToken.PlainText,
		RefreshToken: refreshToken.PlainText,
		ExpiresIn:    int(accessToken.ExpiresAt.Unix()),
		TokenType:    "bearer",
	}

	if err = app.writeJSON(w, http.StatusOK, resp, nil); err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) ResendCodeHandler(w http.ResponseWriter, r *http.Request) {
	var payload ResendCodeRequest
	if err := app.readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	validateEmail(string(payload.Email), v)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
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

		for range 5 {
			if err = app.mailer.Send(string(payload.Email), "user_welcome.tmpl", templateData); err != nil {
				app.logger.Error(err.Error())
			} else {
				app.logger.Info("Successfully sent Welcome email to %s", string(payload.Email))
				break
			}
			time.Sleep(5 * time.Second)
		}
	})

	resp := map[string]string{
		"message": "Email has been sent successfully.",
	}

	if err := app.writeJSON(w, http.StatusOK, resp, nil); err != nil {
		app.serverError(w, r, err)
	}
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

	validateEmail(string(r.Email), v)
	validatePassword(r.Password, v)
}

func validateEmail(email string, v *validator.Validator) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

func validatePassword(password string, v *validator.Validator) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}

func validateLoginRequest(l LoginRequest, v *validator.Validator) {
	validateEmail(string(l.Email), v)
	validatePassword(l.Password, v)
}
