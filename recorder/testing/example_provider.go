// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

package testing

import (
	"time"

	"github.com/alext234/expipe/recorder"
	"github.com/alext234/expipe/tools"
)

// GetRecorder provides a SimpleRecorder for using in the example.
func GetRecorder(url string) *Recorder {
	log := tools.DiscardLogger()
	red, err := New(
		recorder.WithLogger(log),
		recorder.WithEndpoint(url),
		recorder.WithName("recorder_example"),
		recorder.WithIndexName("recorder_example"),
		recorder.WithTimeout(time.Second),
	)
	if err != nil {
		panic(err)
	}
	return red
}
