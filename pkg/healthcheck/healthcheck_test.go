/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package healthcheck

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.impcloud.net/RSP-Inventory-Suite/data-provider-service/app/config"
)

var status = "healthy"

func TestHealthcheck_Healthy(t *testing.T) {
	status = "healthy"
	client := http.DefaultClient
	client.Transport = newMockTransport()
	status := Healthcheck(config.AppConfig.Port)
	if status == 1 {
		t.Error("Healthcheck healthy status should return 0")
	}
}
func TestHealthcheck_Unhealthy(t *testing.T) {
	status = "unhealthy"
	client := http.DefaultClient
	client.Transport = newMockTransport()
	status := Healthcheck(config.AppConfig.Port)
	if status == 0 {
		t.Error("Healthcheck unhealthy status should return 1")
	}

}

type mockTransport struct{}

func newMockTransport() http.RoundTripper {
	return &mockTransport{}
}

// Implement http.RoundTripper
func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	statusCode := 200
	if status == "healthy" {
		statusCode = 200 // http.StatusOK
	} else if status == "unhealthy" {
		statusCode = 500
	}
	// Create mocked http.Response
	response := &http.Response{
		Header:     make(http.Header),
		Request:    req,
		StatusCode: statusCode,
	}
	response.Header.Set("Content-Type", "application/json")
	response.Body = ioutil.NopCloser(strings.NewReader("Service running"))
	return response, nil
}
