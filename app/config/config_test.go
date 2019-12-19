/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package config

import (
	"github.com/intel/rsp-sw-toolkit-im-suite-expect"
	"testing"
)

func TestInitConfig(t *testing.T) {
	w := expect.WrapT(t).StopOnMismatch()
	w.ShouldSucceed(InitConfig())
	w.ShouldNotBeEmptyStr(AppConfig.ServiceName)
}
