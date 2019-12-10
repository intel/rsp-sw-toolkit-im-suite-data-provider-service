/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package healthcheck

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

// Healthcheck performs check to see if server is up and running/responding
func Healthcheck(port string) int {
	resp, err := http.Get("http://127.0.0.1:" + port)
	if err != nil || resp.StatusCode != 200 {
		return 1
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.WithFields(log.Fields{
				"Method": "Healthcheck",
			}).Warning("Failed to close response.")
		}
	}()
	return 0
}
