/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package routes

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.impcloud.net/RSP-Inventory-Suite/data-provider-service/pkg/middlewares"
	"github.impcloud.net/RSP-Inventory-Suite/data-provider-service/pkg/web"
)

// Route struct holds attributes to declare routes
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc web.Handler
}

// Health is used for Docker Healthcheck commands to indicate
// whether the http server is up and running to take requests
// 200 OK
//nolint: unparam
func Health(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {
	web.Respond(ctx, writer, "service running", http.StatusOK)
	return nil
}

// NewRouter creates the routes for GET and POST
func NewRouter() *mux.Router {
	var routes = []Route{
		//swagger:operation GET / default Healthcheck
		//
		// Healthcheck Endpoint
		//
		// Endpoint that is used to determine if the application is ready to take web requests
		//
		// ---
		// consumes:
		// - application/json
		//
		// produces:
		// - application/json
		//
		// schemes:
		// - http
		//
		// responses:
		//   '200':
		//     description: OK
		//
		{
			"Index",
			"GET",
			"/",
			Health,
		},
	}

	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		handler := route.HandlerFunc
		handler = middlewares.Recover(handler)
		handler = middlewares.Logger(handler)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	return router
}
