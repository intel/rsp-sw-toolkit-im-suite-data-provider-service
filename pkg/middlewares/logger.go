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

package middlewares

import (
	"context"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.impcloud.net/Responsive-Retail-Inventory/data-provider-service/pkg/web"
)

// Logger middleware
func Logger(next web.Handler) web.Handler {
	return web.Handler(func(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {

		tracerID := ctx.Value(web.KeyValues).(*web.ContextValues).TraceID
		start := time.Now()
		err := next(ctx, writer, request)

		// Don't log when the index has been accessed since this is what the health check uses and will spam logs when in debug.
		if request.URL.EscapedPath() != "/" {
			log.WithFields(log.Fields{
				"Method":     request.Method,
				"RequestURI": request.RequestURI,
				"Duration":   time.Since(start),
				"TracerId":   tracerID,
			}).Debug("Http Logger middleware")
		}
		// return err since it will contain the error or nil
		return err
	})
}
