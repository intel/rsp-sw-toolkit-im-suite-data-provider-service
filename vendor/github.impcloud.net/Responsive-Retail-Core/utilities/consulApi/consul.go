/*
 * INTEL CONFIDENTIAL
 * Copyright (2018) Intel Corporation.
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

package consulApi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Address  string
	WaitTime time.Duration
}

type Client struct {
	config Config
}

func NewClient(config *Config) (*Client, error) {
	return &Client{config: *config}, nil
}

type QueryOptions struct {
	WaitIndex uint64
	WaitTime  time.Duration
}

type KeyValuePairs []*KeyValuePair

type KeyValuePair struct {
	Key         string
	CreateIndex uint64
	ModifyIndex uint64
	LockIndex   uint64
	Flags       uint64
	Value       []byte
	Session     string
}

func (client *Client) GetValue(key string, queryOptions *QueryOptions) (*KeyValuePair, error) {
	//index=1&wait=600000ms

	endpoint := client.buildEndPoint(key)

	httpClient := &http.Client{
		Timeout: time.Second * 1800,
	}

	request, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create GET http.NewRquest to %s: %s", endpoint, err.Error())
	}
	request.Header.Set("content-type", "application/json;charset=utf-8")

	if queryOptions != nil {
		query := request.URL.Query()
		if queryOptions.WaitIndex != 0 {
			query.Add("index", strconv.FormatUint(queryOptions.WaitIndex, 10))
		}

		if queryOptions.WaitTime != 0 {
			query.Add("wait", durToMsec(queryOptions.WaitTime))
		}

		request.URL.RawQuery = query.Encode()
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("unable to get value for %s: %s", key, err.Error())
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Println(err)
		}
	}()

	if response.StatusCode == http.StatusOK {
		responseData, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading response body for %s: %s", key, err.Error())
		}

		keyValuePairs := KeyValuePairs{}

		if err := json.Unmarshal(responseData, &keyValuePairs); err != nil {
			return nil, fmt.Errorf("unable to unmarshal response for key %s into a KeyValuePairs struct: %s", key, err.Error())
		}

		var keyValuePair *KeyValuePair = nil

		if len(keyValuePairs) > 0 {
			keyValuePair = keyValuePairs[0]
		}

		return keyValuePair, nil
	}

	if response.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	return nil, fmt.Errorf("unable to get value for %s: Received %d status", key, response.StatusCode)
}

func (client *Client) PutValue(key string, value string) error {
	endpoint := client.buildEndPoint(key)

	httpClient := &http.Client{
		Timeout: time.Second * 1800,
	}

	request, err := http.NewRequest("PUT", endpoint, bytes.NewBuffer([]byte(value)))
	if err != nil {
		return fmt.Errorf("unable to create PUT http.NewRquest to %s: %s", endpoint, err.Error())
	}
	request.Header.Set("content-type", "application/json;charset=utf-8")

	response, err := httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("unable to put value for %s: %s", key, err.Error())
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Println(err)
		}
	}()

	if response.StatusCode == http.StatusOK {
		return nil
	}

	return fmt.Errorf("unable to put value for %s: Received %d status", key, response.StatusCode)
}

func (client *Client) DeleteValue(key string) error {
	return nil
}

func durToMsec(dur time.Duration) string {
	ms := dur / time.Millisecond
	if dur > 0 && ms == 0 {
		ms = 1
	}
	return fmt.Sprintf("%dms", ms)
}

func (client *Client) buildEndPoint(key string) string {
	endpoint := client.config.Address + "/" + strings.TrimPrefix(key, "/")
	return endpoint
}
