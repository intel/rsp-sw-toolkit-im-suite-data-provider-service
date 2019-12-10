/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package web

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// JSONError is the response for errors that occur within the API.
//swagger:response internalError
type JSONError struct {
	Error string `json:"error"`
}

var (
	// ErrNotAuthorized occurs when the call is not authorized.
	ErrNotAuthorized = errors.New("Not authorized")

	// ErrDBNotConfigured occurs when the DB is not initialized.
	ErrDBNotConfigured = errors.New("DB not initialized")

	// ErrNotFound is abstracting the mgo not found error.
	ErrNotFound = errors.New("Entity not found")

	// ErrInvalidID occurs when an ID is not in a valid form.
	ErrInvalidID = errors.New("ID is not in it's proper form")

	// ErrValidation occurs when there are validation errors.
	ErrValidation = errors.New("Validation errors occurred")

	// ErrInvalidInput occurs when the input data is invalid
	ErrInvalidInput = errors.New("Invalid input data")
)

// Error handles all error responses for the API.
func Error(ctx context.Context, writer http.ResponseWriter, err error) {

	// Handling client errors
	switch errors.Cause(err) {
	case ErrNotFound:
		RespondError(ctx, writer, err, http.StatusNotFound)
		return

	case ErrInvalidID:
		RespondError(ctx, writer, err, http.StatusBadRequest)
		return

	case ErrValidation:
		RespondError(ctx, writer, err, http.StatusBadRequest)
		return

	case ErrNotAuthorized:
		RespondError(ctx, writer, err, http.StatusUnauthorized)
		return

	case ErrInvalidInput:
		RespondError(ctx, writer, err, http.StatusBadRequest)
		return
	}

	// Handler server error
	contextValues := ctx.Value(KeyValues).(*ContextValues)
	// Log errors
	log.WithFields(log.Fields{
		"Method":     contextValues.Method,
		"RequestURI": contextValues.RequestURI,
		"TracerID":   contextValues.TraceID,
		"Code":       http.StatusInternalServerError,
		"Error":      err.Error(),
	}).Error("Server error")

	//Send a general error to the client
	serverError := errors.New("an error has occurred. Try again")
	RespondError(ctx, writer, serverError, http.StatusInternalServerError)
}

// RespondError sends JSON describing the error
func RespondError(ctx context.Context, writer http.ResponseWriter, err error, code int) {
	Respond(ctx, writer, JSONError{Error: err.Error()}, code)
}

// Respond sends JSON to the client.
// If code is StatusNoContent, v is expected to be nil.
func Respond(ctx context.Context, writer http.ResponseWriter, data interface{}, code int) {

	// Just set the status code and we are done.
	if code == http.StatusNoContent {
		writer.WriteHeader(code)
		return
	}
	if code == http.StatusCreated && data == nil {
		data = "Insert Successful"
	}

	tracerID := ctx.Value(KeyValues).(*ContextValues).TraceID

	// Set the content type.
	writer.Header().Set("Content-Type", "application/json")

	// Write the status code to the response
	writer.WriteHeader(code)

	// Marshal the response data
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.WithFields(log.Fields{
			"Method":   "web.response",
			"Action":   "MarshalIndent",
			"TracerId": tracerID,
			"Error":    err.Error(),
		}).Error("Error Marshalling JSON response")
		jsonData = []byte("{}")
	}

	// Send the result back to the client.
	_, _ = writer.Write(jsonData)
}
