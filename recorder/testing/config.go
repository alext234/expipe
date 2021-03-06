// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

package testing

import (
	"time"

	"github.com/alext234/expipe/recorder"
	"github.com/alext234/expipe/tools"
)

// Config holds the necessary configuration for setting up an elasticsearch
// recorder endpoint.
type Config struct {
	MockName      string
	MockEndpoint  string
	MockTimeout   time.Duration
	MockIndexName string
	MockLogger    tools.FieldLogger
}

// Recorder implements the RecorderConf interface.
func (c *Config) Recorder() (recorder.DataRecorder, error) {
	return New(
		recorder.WithLogger(c.Logger()),
		recorder.WithEndpoint(c.Endpoint()),
		recorder.WithName(c.Name()),
		recorder.WithIndexName(c.IndexName()),
		recorder.WithTimeout(c.Timeout()),
	)
}

// Name is the mocked version.
func (c *Config) Name() string { return c.MockName }

// IndexName is the mocked version.
func (c *Config) IndexName() string { return c.MockIndexName }

// Endpoint is the mocked version.
func (c *Config) Endpoint() string { return c.MockEndpoint }

// Timeout is the mocked version.
func (c *Config) Timeout() time.Duration { return c.MockTimeout }

// Logger is the mocked version.
func (c *Config) Logger() tools.FieldLogger { return c.MockLogger }
