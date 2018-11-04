// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

package testing_test

import (
	"testing"

	rt "github.com/alext234/expipe/recorder/testing"
)

func TestGetRecorderGoodURL(t *testing.T) {
	url := "http://localhost"
	r := rt.GetRecorder(url)
	if r == nil {
		t.Error("r = (nil); want (Recorder)")
	}
	if r.Name() == "" {
		t.Error("r.Name() = (empty); want (string)")
	}
	if r.IndexName() == "" {
		t.Error("r.IndexName() = (empty); want (string)")
	}
	if r.Logger() == nil {
		t.Error("r.Logger() = (nil); want () want (Logger), got (nil)")
	}
	if r.Timeout() <= 0 {
		t.Errorf("r.Timeout() = (%d); want (>1s)", r.Timeout())
	}
	url = "bad url"
	var panicked bool
	func() {
		defer func() {
			if e := recover(); e != nil {
				panicked = true
			}
		}()
		rt.GetRecorder(url)
		if !panicked {
			t.Error("panic = (false); want (true): didn't panic on bad url")
		}
	}()
}
