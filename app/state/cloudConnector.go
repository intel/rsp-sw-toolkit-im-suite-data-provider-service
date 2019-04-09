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
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type cloudConnAuth struct {
	AuthType string `json:"authtype"` // "oauth2"
	Endpoint string `json:"endpoint"` // oauth endpoint
	Data     string `json:"data"`     // username:password
}

type cloudConnRequest struct {
	URL    string        `json:"url"`    // destination
	Method string        `json:"method"` // GET
	Auth   cloudConnAuth `json:"auth"`   // above struct
}

type cloudConnRequestNoAuth struct {
	URL    string `json:"url"`    // destination
	Method string `json:"method"` // GET
}

type cloudConnResponse struct {
	Body       []byte
	StatusCode int
	Header     http.Header
}

// IsProxied returns true if the FileCache is proxying requests through the
// CloudConnector.
func (ccr *CCRequest) IsProxied() bool {
	return ccr.isProxied
}

// ProxyThroughCloudConnector sets the package to proxy all downloads through
// the configured cloud connector URL, using the given oauth URL and credentials.
func (ccr *CCRequest) ProxyThroughCloudConnector(ccURL, oauthURL, creds string) {
	logrus.Infof("Enabling cloud connector at %s for OAuth2 server %s",
		ccURL, oauthURL)
	auth := cloudConnAuth{AuthType: "oauth2", Endpoint: oauthURL, Data: creds}

	ccr.isProxied = true

	// replace the downloadFunc with a POST to the cloud connector endpoint
	ccr.requestFunc = func(client *http.Client, source string, useAuth bool) (*http.Response, error) {
		logrus.Debugf("Proxying request for %s through cloud connector", source)

		var request []byte
		var err error

		logrus.Debugf("Use auth: %t", useAuth)
		if useAuth == true {
			request, err = json.Marshal(cloudConnRequest{URL: source, Method: "GET", Auth: auth})
		} else {
			request, err = json.Marshal(cloudConnRequestNoAuth{URL: source, Method: "GET"})
		}

		if err != nil {
			return nil, errors.Wrapf(err, "unable to create cloud connector request")
		}
		logrus.Debugf("Request to cloudconnector: %s", request)

		resp, err := client.Post(ccURL, "application/json", bytes.NewBuffer(request))
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusNoContent {
			err = errors.New(error204)
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			err = errors.Errorf("Cloud Connector returned non-200 status: %d.",
				resp.StatusCode)
			return nil, err
		}
		if resp.Body == nil {
			err = errors.Errorf("No response body from cloud connector")
			return nil, err
		}

		ccBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to read cloud response body")
		}

		var ccResp cloudConnResponse
		if err := json.Unmarshal(ccBody, &ccResp); err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: ccResp.StatusCode,
			Body:       ioutil.NopCloser(bytes.NewBuffer(ccResp.Body)),
			Header:     ccResp.Header,
		}, nil
	}
}
