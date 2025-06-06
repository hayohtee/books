package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/hayohtee/books/internal/data"
	"github.com/hayohtee/books/internal/validator"
	openapitypes "github.com/oapi-codegen/runtime/types"
	"math"
	"net/http"
	"strings"
)

func (app *application) ListBookHandler(w http.ResponseWriter, r *http.Request, params ListBookHandlerParams) {
	userID, err := app.contextGetUserID(r)
	if err != nil {
		app.errorResponse(w, r, http.StatusUnauthorized, Error{Message: "User is not authorized"})
		return
	}

	v := validator.New()
	validateListBookParams(params, v)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	var searchName string
	if params.Name != nil {
		searchName = *params.Name
	}
	var page = 1
	if params.Page != nil {
		page = *params.Page
	}
	var pageSize = 10
	if params.PageSize != nil {
		pageSize = *params.PageSize
	}

	rows, err := app.queries.ListBookForUser(r.Context(), data.ListBookForUserParams{
		UserID:         userID,
		PlaintoTsquery: searchName,
		Limit:          int32(pageSize),
		Offset:         int32((page - 1) * pageSize),
	})

	if err != nil {
		app.serverError(w, r, err)
		return
	}

	var totalRecords int64
	var books []BookResponse
	for _, value := range rows {
		totalRecords = value.TotalRecords
		book := BookResponse{
			Id:        value.ID,
			Name:      value.Name,
			UserId:    value.UserID,
			CreatedAt: value.CreatedAt,
			UpdatedAt: value.UpdatedAt,
		}
		books = append(books, book)
	}

	if books == nil {
		books = make([]BookResponse, 0)
	}

	pagination := calculateMetadata(int(totalRecords), page, pageSize)
	resp := ListBookResponse{
		Metadata: pagination,
		Items:    books,
	}

	if err := app.writeJSON(w, http.StatusOK, resp, nil); err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) CreateBookHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := app.contextGetUserID(r)
	if err != nil {
		app.errorResponse(w, r, http.StatusUnauthorized, Error{Message: "User is not authorized"})
		return
	}

	var payload CreateBookRequest
	if err := app.readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	v.Check(payload.Name != "", "name", "must be provided")
	v.Check(len(payload.Name) <= 500, "name", "must not be more than 500 bytes")
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	row, err := app.queries.CreateBook(r.Context(), data.CreateBookParams{
		UserID: userID,
		Name:   payload.Name,
	})

	if err != nil {
		switch {
		case strings.Contains(err.Error(), "books_user_id_name_key"):
			app.errorResponse(w, r, http.StatusConflict, Error{Message: "Book already exists"})
		default:
			app.serverError(w, r, err)
		}
		return
	}

	header := make(http.Header)
	header.Set("Location", fmt.Sprintf("/books/%s", row.ID))

	resp := BookResponse{
		Id:        row.ID,
		Name:      payload.Name,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		UserId:    userID,
	}

	if err := app.writeJSON(w, http.StatusCreated, resp, header); err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) DeleteBookHandler(w http.ResponseWriter, r *http.Request, id openapitypes.UUID) {

}

func (app *application) GetBookHandler(w http.ResponseWriter, r *http.Request, id openapitypes.UUID) {
	userID, err := app.contextGetUserID(r)
	if err != nil {
		app.errorResponse(w, r, http.StatusUnauthorized, Error{Message: "User is not authorized"})
		return
	}

	book, err := app.queries.GetBook(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			app.notFoundResponse(w, r)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	if userID.String() != book.UserID.String() {
		app.notPermittedResponse(w, r)
		return
	}

	resp := BookResponse{
		Id:        book.ID,
		Name:      book.Name,
		CreatedAt: book.CreatedAt,
		UpdatedAt: book.UpdatedAt,
		UserId:    book.UserID,
	}

	if err := app.writeJSON(w, http.StatusOK, resp, nil); err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) UpdateBookHandler(w http.ResponseWriter, r *http.Request, id openapitypes.UUID) {
	userID, err := app.contextGetUserID(r)
	if err != nil || userID == uuid.Nil {
		app.authenticationRequiredResponse(w, r)
		return
	}

	var payload UpdateBookRequest
	if err := app.readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	v.Check(payload.Name != "", "name", "must be provided")
	v.Check(len(payload.Name) <= 500, "name", "must not be more than 500 bytes")
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	book, err := app.queries.GetBook(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			app.notFoundResponse(w, r)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	if userID.String() != book.UserID.String() {
		app.notPermittedResponse(w, r)
		return
	}

	book, err = app.queries.UpdateBook(r.Context(), data.UpdateBookParams{
		Name:    payload.Name,
		ID:      book.ID,
		Version: book.Version,
		UserID:  userID,
	})

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			app.editConflictResponse(w, r)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	resp := BookResponse{
		Id:        book.ID,
		Name:      book.Name,
		CreatedAt: book.CreatedAt,
		UpdatedAt: book.UpdatedAt,
		UserId:    book.UserID,
	}

	if err := app.writeJSON(w, http.StatusOK, resp, nil); err != nil {
		app.serverError(w, r, err)
	}
}

func validateListBookParams(params ListBookHandlerParams, v *validator.Validator) {
	if params.Page != nil {
		v.Check(*params.Page > 0, "page", "must be greater than zero")
	}
	if params.PageSize != nil {
		v.Check(*params.PageSize > 0, "page_size", "must be greater than zero")
		v.Check(*params.PageSize <= 100, "page_size", "must be a maximum of 100")
	}
	if params.Name != nil {
		v.Check(len(*params.Name) <= 500, "name", "must be less than 500 characters")
	}
}

// calculateMetadata calculates the appropriate pagination metadata
// values given the total number of records, current page, and page size values.
func calculateMetadata(totalRecords, page, pageSize int) Pagination {
	if totalRecords == 0 {
		return Pagination{}
	}

	return Pagination{
		CurrentPage: page,
		PageSize:    pageSize,
		FirstPage:   1,
		LastPage:    int(math.Ceil(float64(totalRecords) / float64(pageSize))),
		TotalItems:  totalRecords,
	}
}
