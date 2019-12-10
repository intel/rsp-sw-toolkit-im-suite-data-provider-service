/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package config

import (
	"github.impcloud.net/RSP-Inventory-Suite/expect"
	"testing"
)

func TestInitConfig(t *testing.T) {
	w := expect.WrapT(t).StopOnMismatch()
	w.ShouldSucceed(InitConfig())
	w.ShouldNotBeEmptyStr(AppConfig.ServiceName)
}
