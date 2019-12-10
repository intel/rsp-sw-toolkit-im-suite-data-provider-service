/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"flag"
	"os"

	"github.impcloud.net/RSP-Inventory-Suite/data-provider-service/pkg/healthcheck"
	"github.impcloud.net/RSP-Inventory-Suite/utilities/go-metrics"

	"github.com/sirupsen/logrus"
)

func exitIfError(err error, errorGauge metrics.Gauge, args ...interface{}) {
	if err == nil {
		return
	}

	errorGauge.Update(1)
	logrus.WithError(err).
		WithFields(logrus.Fields{
			"Method": "main",
		}).Fatal(args...)
}

func healthCheck(port string) {
	isHealthyPtr := flag.Bool("isHealthy", false, "a bool, runs a healthcheck")
	flag.Parse()

	if *isHealthyPtr {
		status := healthcheck.Healthcheck(port)
		os.Exit(status)
	}
}
