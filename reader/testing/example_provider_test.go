// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

package testing_test

import (
	"testing"

	rt "github.com/alext234/expipe/reader/testing"
)

func TestGetRecorderGoodURL(t *testing.T) {
	url := "http://localhost"
	r := rt.GetReader(url)
	if r == nil {
		t.Error("r = (nil); want (Recorder)")
	}
	if r.Name() == "" {
		t.Error("r.Name(): Name cannot be empty")
	}
	if r.TypeName() == "" {
		t.Error("r.TypeName(): TypeName cannot be empty")
	}
	if r.Logger() == nil {
		t.Error("r.Logger() = (nil); want (Logger)")
	}
	if r.Timeout() <= 0 {
		t.Errorf("r.Timeout() = (%d); want (>1s)", r.Timeout())
	}
	if r.Interval() == 0 {
		t.Error("r.Interval() = (0); want (>1s)")
	}
	url = "bad url"
	var panicked bool
	func() {
		defer func() {
			if e := recover(); e != nil {
				panicked = true
			}
		}()
		rt.GetReader(url)
		if !panicked {
			t.Error("panicked = (false); want (true)")
		}
	}()
}
