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

package configuration

import (
	"fmt"
	"github.impcloud.net/Responsive-Retail-Core/utilities/consulApi"
	"log"
	"time"
)

const (
	watchTimeout = time.Minute * 10
)

type Watcher struct {
	watchKey string
	consul   *consulApi.Client
}

func NewWatcher(consul *consulApi.Client, key string) (*Watcher, error) {
	if consul == nil {
		return nil, fmt.Errorf("consul can not be nil")
	}

	if key == "" {
		return nil, fmt.Errorf("key can not be empty")
	}

	watcher := Watcher{
		consul:   consul,
		watchKey: key,
	}

	return &watcher, nil
}

func (watcher *Watcher) Start(changeCallback func([]byte)) error {
	keyValuePair, err := watcher.consul.GetValue(watcher.watchKey, nil)
	if err != nil {
		return fmt.Errorf("unable to GET watch key (%s) data: %s", watcher.watchKey, err.Error())
	}

	if keyValuePair == nil {
		return fmt.Errorf("unable to GET watch key (%s) data: key is not found", watcher.watchKey)
	}

	go func(targetIndex uint64) {

		queryOptions := consulApi.QueryOptions{
			WaitIndex: targetIndex,
			WaitTime:  watchTimeout,
		}

		for {
			keyValuePair, err = watcher.consul.GetValue(watcher.watchKey, &queryOptions)
			if err != nil {
				log.Printf("Error watching %s key: %s", watcher.watchKey, err.Error())
				time.Sleep(time.Second * 10) // Assume consul is restarting, so want long wait.
				continue
			}

			if keyValuePair.ModifyIndex == targetIndex {
				// No change , so must have timed out. Try again
				continue
			}

			changeCallback(keyValuePair.Value)

			// This is required so we block waiting for the next change
			targetIndex = keyValuePair.ModifyIndex
		}
	}(keyValuePair.ModifyIndex)

	return nil
}
