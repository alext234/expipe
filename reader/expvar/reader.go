// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

// Package expvar contains logic to read from an expvar provide. The data comes
// in JSON format. The GC and memory related information will be changed to
// better presented to the data recorders. Bytes will be turned into megabytes,
// gc lists will be truncated to remove zero values.
package expvar

import (
	"bytes"
	"context"
	"net/url"
	"time"

	"github.com/alext234/expipe/datatype"
	"github.com/alext234/expipe/reader"
	"github.com/alext234/expipe/tools"
	"github.com/alext234/expipe/tools/token"

	"github.com/pkg/errors"
	"golang.org/x/net/context/ctxhttp"
)

// Reader can read from any application that exposes expvar information.
// It implements DataReader interface.
type Reader struct {
	name     string
	endpoint string
	log      tools.FieldLogger
	mapper   datatype.Mapper
	typeName string
	interval time.Duration
	timeout  time.Duration
	pinged   bool
}

// New generates the Reader based on the provided options.
func New(options ...func(reader.Constructor) error) (*Reader, error) {
	r := &Reader{}
	for _, op := range options {
		err := op(r)
		if err != nil {
			return nil, errors.Wrap(err, "option creation")
		}
	}

	if r.name == "" {
		return nil, reader.ErrEmptyName
	}
	if r.endpoint == "" {
		return nil, reader.ErrEmptyEndpoint
	}
	if r.mapper == nil {
		r.mapper = datatype.DefaultMapper()
	}
	if r.typeName == "" {
		r.typeName = r.name
	}
	if r.interval == 0 {
		r.interval = time.Second
	}
	if r.timeout == 0 {
		r.timeout = 5 * time.Second
	}
	if r.log == nil {
		r.log = tools.GetLogger("error")
	}
	r.log = r.log.WithField("engine", "expipe")
	return r, nil
}

// Ping pings the endpoint and return nil if was successful.
// It returns an EndpointNotAvailableError if the endpoint id unavailable.
func (r *Reader) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	_, err := ctxhttp.Head(ctx, nil, r.endpoint)
	if err != nil {
		return reader.EndpointNotAvailableError{Endpoint: r.endpoint, Err: err}
	}
	r.pinged = true
	return nil
}

// Read begins reading from the target. It returns an error back to the engine
// if it can't read from metrics provider, Ping() is not called or the endpoint
// has been unresponsive too many times.
func (r *Reader) Read(job *token.Context) (*reader.Result, error) {
	if !r.pinged {
		return nil, reader.ErrPingNotCalled
	}
	resp, err := ctxhttp.Get(job, nil, r.endpoint)

	if err != nil {
		if _, ok := err.(*url.Error); ok {
			err = reader.EndpointNotAvailableError{Endpoint: r.endpoint, Err: err}
		}
		r.log.WithField("reader", "expvar_reader").
			WithField("name", r.Name()).
			WithField("ID", job.ID()).
			Debugf("%s: error making request: %v", r.name, err)
		return nil, err
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading buffer")
	}
	content := buf.Bytes()
	if !tools.IsJSON(content) {
		return nil, reader.ErrInvalidJSON
	}
	res := &reader.Result{
		ID:       job.ID(),
		Time:     time.Now(), // It is sensible to record the time now
		Content:  content,
		TypeName: r.TypeName(),
		Mapper:   r.Mapper(),
	}
	return res, nil
}

// Name shows the name identifier for this reader.
func (r *Reader) Name() string { return r.name }

// SetName sets the name of the reader.
func (r *Reader) SetName(name string) { r.name = name }

// Endpoint returns the endpoint.
func (r *Reader) Endpoint() string { return r.endpoint }

// SetEndpoint sets the endpoint of the reader.
func (r *Reader) SetEndpoint(endpoint string) { r.endpoint = endpoint }

// TypeName shows the typeName the recorder should record as.
func (r *Reader) TypeName() string { return r.typeName }

// SetTypeName sets the type name of the reader.
func (r *Reader) SetTypeName(typeName string) { r.typeName = typeName }

// Mapper returns the mapper object.
func (r *Reader) Mapper() datatype.Mapper { return r.mapper }

// SetMapper sets the mapper of the reader.
func (r *Reader) SetMapper(mapper datatype.Mapper) { r.mapper = mapper }

// Interval returns the interval.
func (r *Reader) Interval() time.Duration { return r.interval }

// SetInterval sets the interval of the reader.
func (r *Reader) SetInterval(interval time.Duration) { r.interval = interval }

// Timeout returns the time-out.
func (r *Reader) Timeout() time.Duration { return r.timeout }

// SetTimeout sets the timeout of the reader.
func (r *Reader) SetTimeout(timeout time.Duration) { r.timeout = timeout }

// SetLogger sets the log of the reader.
func (r *Reader) SetLogger(log tools.FieldLogger) { r.log = log }
