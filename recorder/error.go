// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

package recorder

import "fmt"

var (
	// ErrEmptyName is the error when the package name is empty
	ErrEmptyName = fmt.Errorf("name cannot be empty")

	// ErrEmptyEndpoint is the error when the given endpoint is empty
	ErrEmptyEndpoint = fmt.Errorf("endpoint cannot be empty")

	// ErrEmptyIndexName is the error when the index_name is an empty string
	ErrEmptyIndexName = fmt.Errorf("index_name cannot be empty")
)

// ErrInvalidEndpoint is the error when the endpoint is not a valid url
type ErrInvalidEndpoint string

// InvalidEndpoint defines the behaviour of the error
func (ErrInvalidEndpoint) InvalidEndpoint() {}
func (e ErrInvalidEndpoint) Error() string  { return fmt.Sprintf("invalid endpoint: %s", string(e)) }

// ErrLowBackoffValue is the error when the endpoint is not a valid url
type ErrLowBackoffValue int64

// LowBackoffValue defines the behaviour of the error
func (ErrLowBackoffValue) LowBackoffValue() {}
func (e ErrLowBackoffValue) Error() string  { return fmt.Sprintf("back off should be at least 5: %d", e) }

// ErrParseTimeOut is for when the timeout cannot be parsed
type ErrParseTimeOut struct {
	Timeout string
	Err     error
}

// ParseTimeOut defines the behaviour of the error
func (ErrParseTimeOut) ParseTimeOut() {}
func (e ErrParseTimeOut) Error() string {
	return fmt.Sprintf("parse timeout (%s): %s", e.Timeout, e.Err)
}