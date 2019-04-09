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

package state

import (
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// error204 is returned if the server returns 204 (No Content) during a download.
const error204 = "the server returned Status 204, No Content"

// CCRequest represents the request performed through the cloud connector
// to GET JSON payload from a cloud endpoint
type CCRequest struct {
	retries uint
	client  *http.Client
	// requestFunc is the function used when the ccRequest performs a GET request
	// through the CloudConnector
	requestFunc func(client *http.Client, source string, useAuth bool) (*http.Response, error)
	isProxied   bool
}

// NewCCRequest creates a new CloudConnector Request component to perform GET requests
func NewCCRequest(timeout int, retries uint) (ccr *CCRequest) {
	if timeout < 0 {
		timeout = 0
	}

	ccr = &CCRequest{
		retries:   retries,
		isProxied: false,
		client:    &http.Client{Timeout: time.Duration(timeout) * time.Second},
	}

	return
}

// SetClient replaces the http.Client used by this CCRequest
func (ccr *CCRequest) SetClient(client *http.Client) {
	ccr.client = client
}

// PerformRequest issues a GET request to the specified destination and pushes the data into
// the ItemChannel that will push the response into the
func (ccr *CCRequest) PerformRequest(source string, useAuth bool) (jsonResponse []byte, err error) {
	logrus.Debugf("Attempting to send GET request file to %s.", source)

	var resp *http.Response
	if ccr.IsProxied() {
		resp, err = ccr.requestFunc(ccr.client, source, useAuth)
	} else {
		resp, err = ccr.client.Get(source)
	}

	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNoContent {
		err = errors.New(error204)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		err = errors.Errorf("GET returned non-200 status: %d.", resp.StatusCode)
		return nil, err
	}
	if resp.Body == nil {
		err = errors.Errorf("No response body from server")
		return nil, err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
		jsonResponse, err = ioutil.ReadAll(resp.Body)
	}

	logrus.Debugf("JSON response: %s", jsonResponse)
	return jsonResponse, nil
}

// downloadDirectly just issues a GET request directly to the source.
func downloadDirectly(client *http.Client, source string) (*http.Response, error) {
	return client.Get(source)
}
