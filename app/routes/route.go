/*
 * INTEL CONFIDENTIAL
 * Copyright (2019) Intel Corporation.
 *
 * The source code contained or described herein and all documents related to the source code ("Material")
 * are owned by Intel Corporation or its suppliers or licensors. Title to the Material remains with
 * Intel Corporation or its suppliers and licensors. The Material may contain trade secrets and proprietary
 * and confidential information of Intel Corporation and its suppliers and licensors, and is protected by
 * worldwide copyright and trade secret laws and treaty provisions. No part of the Material may be used,
 * copied, reproduced, modified, published, uploaded, posted, transmitted, distributed, or disclosed in
 * any way without Intel/'s prior express written permission.
 * No license under any patent, copyright, trade secret or other intellectual property right is granted
 * to or conferred upon you by disclosure or delivery of the Materials, either expressly, by implication,
 * inducement, estoppel or otherwise. Any license under such intellectual property rights must be express
 * and approved by Intel in writing.
 * Unless otherwise agreed by Intel in writing, you may not remove or alter this notice or any other
 * notice embedded in Materials by Intel or Intel's suppliers or licensors in any way.
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
