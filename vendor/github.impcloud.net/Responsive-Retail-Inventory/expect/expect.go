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

// Package expect makes tests' expectations more obvious.
package expect

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// TWrapper wraps a *testing.T, allowing fluent asserts with clear error messages.
// Any *testing.T methods, such as t.Log, can be used as normal.
type TWrapper struct {
	*testing.T
	onFail failFunc
}

// WrapT returns a TWrapper, which allows several fluent assertions while testing.
//noinspection GoUnusedExportedFunction
func WrapT(t *testing.T) *TWrapper {
	return (&TWrapper{T: t}).ContinueOnMismatch()
}

type failFunc = func(fmt string, args ...interface{})
func (t *TWrapper) dup(ff failFunc) *TWrapper {
	return &TWrapper{
		T:      t.T,
		onFail: ff,
	}
}

// As adds d to the message if an expectation fails. Use it with statements like:
//     w.As("empty json input").ShouldFail(validateJSON(""))
func (t *TWrapper) As(d interface{}) *TWrapper {
	t.Helper()
	t2 := t.dup(func(format string, args ...interface{}) {
		t.Helper()
		t.onFail(fmt.Sprintf("Failure for '%+v': %s", d, format), args...)
	})
	return t2
}

// StopOnMismatch returns a TWrapper that stop testing as soon as an expectation fails.
func (t *TWrapper) StopOnMismatch() *TWrapper {
	t.Helper()
	return t.dup(func(fmt string, args ...interface{}) {
		t.Helper()
		t.Fatalf(fmt, args...)
	})
}

// ContinueOnMismatch returns a TWrapper that continues testing, even if an expectation fails.
func (t *TWrapper) ContinueOnMismatch() *TWrapper {
	t.Helper()
	return t.dup(func(fmt string, args ...interface{}) {
		t.Helper()
		t.Errorf(fmt, args...)
	})
}

// ShouldBeEqual expects x and y to be equal, as compared by reflect.DeepEqual.
func (t *TWrapper) ShouldBeEqual(x, y interface{}) {
	t.Helper()
	if reflect.DeepEqual(x, y) {
		return
	}

	t.onFail("%+v != %+v (types %T, %T)", x, y, x, y)
}

// ShouldNotBeEqual expects x and y to be unequal as compared by reflect.DeepEqual.
func (t *TWrapper) ShouldNotBeEqual(x, y interface{}) {
	t.Helper()
	if !reflect.DeepEqual(x, y) {
		return
	}

	t.onFail("expected unequal, but were both: %+v", x, y)
}

// ShouldBeSameType expects x and y to be the same type.
func (t *TWrapper) ShouldBeSameType(x, y interface{}) {
	t.Helper()
	if x == nil || y == nil && x != y ||
		reflect.ValueOf(x).Type() != reflect.ValueOf(y).Type() {
		t.onFail("type mismatch: (%T, %T) with values (%v, %v)",
			x, y, x, y)
	}
}

// ShouldBeFalse expects v to be false.
func (t *TWrapper) ShouldBeFalse(v interface{}) {
	t.Helper()
	t.ShouldBeEqual(false, v)
}

// ShouldBeFalse expects v to be true.
func (t *TWrapper) ShouldBeTrue(v interface{}) {
	t.Helper()
	t.ShouldBeEqual(true, v)
}

func isNil(v interface{}) bool {
	switch reflect.ValueOf(v).Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
		return v == nil || reflect.ValueOf(v).IsNil()
	default:
		return v == nil || reflect.TypeOf(v) == nil
	}
}

// ShouldNotBeNil expect v to not be nil.
func (t *TWrapper) ShouldNotBeNil(v interface{}) {
	t.Helper()
	if !isNil(v) {
		return
	}
	t.onFail("value is nil, but shouldn't be (type %T)", v)
}

// ShouldBeNil expect v to be nil.
func (t *TWrapper) ShouldBeNil(v interface{}) {
	t.Helper()
	if isNil(v) {
		return
	}
	t.onFail("expected value to be nil, but it's %+v (type %T)", v, v)
}

// ShouldBeEmptyStr expects v to be an empty string.
func (t *TWrapper) ShouldBeEmptyStr(v interface{}) {
	t.Helper()
	t.ShouldBeEqual("", v)
}

// ShouldNotBeEmptyStr expect v not to be an empty string.
func (t *TWrapper) ShouldNotBeEmptyStr(v interface{}) {
	t.Helper()
	t.ShouldNotBeEqual("", v)
}

// ShouldContainStr expects string x to contain y.
func (t *TWrapper) ShouldContainStr(x, y string) {
	t.Helper()
	t.As(fmt.Sprintf("'%s' should contain '%s'", x, y)).ShouldBeTrue(strings.Contains(x, y))
}

// ShouldFail expects a given err to not be nil; it can be used with the immediate
// result of a function, such as w.ShouldFail(funcThatReturnsError()).
func (t *TWrapper) ShouldFail(err error) error {
	t.Helper()
	if err == nil {
		t.onFail("expected an error, but none occurred")
	}
	return err
}

// ShouldSucceed expects a given err to be nil; it can be used with the immediate
// result of a function, such as w.ShouldSucceed(funcThatReturnsError()).
func (t *TWrapper) ShouldSucceed(err error) {
	t.Helper()
	if err != nil {
		t.onFail("an unexpected error occurred: %+v", err)
	}
}

// ShouldSucceedLater takes a function, which it calls using ShouldSucceed.
// It's meant to be used for deferred functions, such as:
//     defer w.ShouldSucceedLater(c.Close)
// which will defer w.ShouldSucceed(c.Close()), ensuring that c.Close() succeeds.
func (t *TWrapper) ShouldSucceedLater(f func() error) {
	t.Helper()
	t.ShouldSucceed(f())
}

// ShouldHaveError takes a result interface and error and expects that the error
// is not nil; it can be used with the immediate result of a call that's expected
// to fail, but has a return value:
//     w.ShouldHaveError(os.Open("missingFile"))
func (t *TWrapper) ShouldHaveError(result interface{}, err error) error {
	t.Helper()
	if err == nil {
		t.onFail("expected an error, but none occurred")
	}

	if result != nil {
		v := reflect.ValueOf(result)
		if v.Kind() != reflect.Ptr || !v.IsNil() {
			t.Logf("function returned result: %+v", result)
		}
	}
	return err
}

// ShouldHaveResult takes a result interface and an error and expects that the
// error is not nil, in which case it returns the result, unmodified. This allows
// single-line error validation on function calls that return a type and an error.
// Note that, because the result must be wrapped in an interface, you'll have to
// type assert it before using it:
//    f := w.ShouldHaveResult(os.Open("myFile")).(*os.File)
func (t *TWrapper) ShouldHaveResult(result interface{}, err error) interface{} {
	t.Helper()
	if err != nil {
		t.StopOnMismatch().onFail("an unexpected error occurred: %+v", err)
	}
	return result
}

// ShouldHaveLength expects the given interface to have a particular length.
// The input can be an Array, Channel, Map, Slice, or String; if it isn't, then
// the check automatically fails.
func (t *TWrapper) ShouldHaveLength(itemWithLen interface{}, length int) {
	t.Helper()
	v := reflect.ValueOf(itemWithLen)
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		if v.Len() == length {
			break
		}
		t.onFail("expected length %d, but actual is %d (type: %T, value: %+v)",
			length, v.Len(), itemWithLen, itemWithLen)
	default:
		t.onFail("expected array, channel, map, slice, or string, but was %T (%+v)",
			itemWithLen, v.Kind())
	}
}
