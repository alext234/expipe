// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

// Package self contains codes for recording expipe's own metrics.
//
package self

import (
	"bytes"
	"context"
	"expvar"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/alext234/expipe/datatype"
	"github.com/alext234/expipe/reader"
	"github.com/alext234/expipe/tools"
	"github.com/alext234/expipe/tools/token"
	"github.com/pkg/errors"
	"golang.org/x/net/context/ctxhttp"
)

// Reader reads from expipe own application's metric information.
// It implements DataReader interface.
type Reader struct {
	name       string
	typeName   string
	log        tools.FieldLogger
	mapper     datatype.Mapper
	interval   time.Duration
	timeout    time.Duration
	quit       chan struct{}
	endpoint   string
	pinged     bool
	testMode   bool // this is for internal tests. You should not set it to true.
	tempServer *httptest.Server
}

// New exposes expipe's own metrics.
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
	r.log = r.log.WithField("engine", "self")
	r.quit = make(chan struct{})
	return r, nil
}

// Ping pings the endpoint and return nil if was successful. It returns an error
// if the endpoint is not available.
// TODO: this method is duplicated. Create a Pinger type and share the logic.
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

// Read send the metrics back. The error is usually nil.
func (r *Reader) Read(job *token.Context) (*reader.Result, error) {
	if !r.pinged {
		return nil, reader.ErrPingNotCalled
	}
	// To support the tests
	if r.testMode {
		_, err := r.readMetricsFromURL(job)
		if err != nil {
			return nil, err
		}
		// to support invalid JSON check in tests
		if !checkJSON(job, r.endpoint) {
			return nil, reader.ErrInvalidJSON
		}
	}
	buf := new(bytes.Buffer) // construct a json encoder and pass it
	fmt.Fprint(buf, "{\n")
	first := true
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			fmt.Fprint(buf, ",\n")
		}
		first = false
		fmt.Fprintf(buf, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprint(buf, "\n}\n")
	res := &reader.Result{
		ID:       job.ID(),
		Time:     time.Now(), // It is sensible to record the time now
		Content:  buf.Bytes(),
		TypeName: r.TypeName(),
		Mapper:   r.Mapper(),
	}
	return res, nil
}

func checkJSON(job context.Context, endpoint string) bool {
	resp, err := ctxhttp.Get(job, nil, endpoint)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	content := buf.Bytes()
	if !tools.IsJSON(content) {
		return false
	}
	return true
}

// Name shows the name identifier for this reader
func (r *Reader) Name() string { return r.name }

// SetName sets the name of the reader
func (r *Reader) SetName(name string) { r.name = name }

// Endpoint returns the endpoint
func (r *Reader) Endpoint() string { return r.endpoint }

// SetEndpoint sets the endpoint of the reader
func (r *Reader) SetEndpoint(endpoint string) { r.endpoint = endpoint }

// TypeName shows the typeName the recorder should record as
func (r *Reader) TypeName() string { return r.typeName }

// SetTypeName sets the type name of the reader
func (r *Reader) SetTypeName(typeName string) { r.typeName = typeName }

// Mapper returns the mapper object
func (r *Reader) Mapper() datatype.Mapper { return r.mapper }

// SetMapper sets the mapper of the reader
func (r *Reader) SetMapper(mapper datatype.Mapper) { r.mapper = mapper }

// Interval returns the interval
func (r *Reader) Interval() time.Duration { return r.interval }

// SetInterval sets the interval of the reader
func (r *Reader) SetInterval(interval time.Duration) { r.interval = interval }

// Timeout returns the time-out
func (r *Reader) Timeout() time.Duration { return r.timeout }

// SetTimeout sets the timeout of the reader
func (r *Reader) SetTimeout(timeout time.Duration) { r.timeout = timeout }

// SetLogger sets the log of the reader
func (r *Reader) SetLogger(log tools.FieldLogger) { r.log = log }

// SetTestMode sets the mode to testing for testing purposes
// This is because the way self works
func (r *Reader) SetTestMode() { r.testMode = true }

// this is only used in tests.
// TODO: [refactor] this.
func (r *Reader) readMetricsFromURL(job *token.Context) (*reader.Result, error) {
	resp, err := http.Get(r.endpoint)
	if err != nil {
		if _, ok := err.(*url.Error); ok {
			err = reader.EndpointNotAvailableError{Endpoint: r.endpoint, Err: err}
		}
		r.log.WithField("reader", "self").
			WithField("ID", job.ID()).
			// Error because it is a local dependency.
			Errorf("%s: error making request: %v", r.name, err)
		return nil, err
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	res := &reader.Result{
		ID:       job.ID(),
		Time:     time.Now(), // It is sensible to record the time now
		Content:  buf.Bytes(),
		TypeName: r.TypeName(),
		Mapper:   r.Mapper(),
	}
	return res, nil
}

// WithTempServer creates a temp server that does nothing and attach it to the
// reader to response to Engine pings.
func WithTempServer() func(reader.Constructor) error {
	return func(e reader.Constructor) error {
		if sl, ok := e.(*Reader); ok {
			sl.tempServer = httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {}),
			)
			sl.endpoint = sl.tempServer.URL
			return nil
		}
		return errors.New("incompatible server")
	}
}
