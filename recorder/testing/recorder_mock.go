// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

package testing

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/arsham/expvastic/lib"
	"github.com/arsham/expvastic/recorder"
)

// SimpleRecorder is designed to be used in tests
type SimpleRecorder struct {
	name       string
	endpoint   string
	indexName  string
	log        logrus.FieldLogger
	timeout    time.Duration
	ErrorFunc  func() error
	backoff    int
	strike     int
	Smu        sync.RWMutex
	RecordFunc func(context.Context, *recorder.RecordJob) error
}

// NewSimpleRecorder returns a SimpleRecorder instance
func NewSimpleRecorder(ctx context.Context, log logrus.FieldLogger, name, endpoint, indexName string, timeout time.Duration, backoff int) (*SimpleRecorder, error) {
	if backoff < 5 {
		return nil, recorder.ErrLowBackoffValue(backoff)
	}
	url, err := lib.SanitiseURL(endpoint)
	if err != nil {
		return nil, recorder.ErrInvalidEndpoint(endpoint)
	}
	w := &SimpleRecorder{
		name:      name,
		endpoint:  url,
		indexName: indexName,
		log:       log,
		timeout:   timeout,
		backoff:   backoff,
	}
	return w, nil
}

// Record calls the RecordFunc if exists, otherwise continues as normal
func (s *SimpleRecorder) Record(ctx context.Context, job *recorder.RecordJob) error {
	s.Smu.RLock()
	if s.RecordFunc != nil {
		s.Smu.RUnlock()
		return s.RecordFunc(ctx, job)
	}
	s.Smu.RUnlock()
	if s.strike > s.backoff {
		return recorder.ErrBackoffExceeded
	}
	res, err := http.Get(s.endpoint)
	if err != nil {
		if v, ok := err.(*url.Error); ok {
			if strings.Contains(v.Error(), "getsockopt: connection refused") {
				s.strike++
			}
		}
		return err
	}
	res.Body.Close()
	return nil
}

// Name returns the name
func (s *SimpleRecorder) Name() string { return s.name }

// IndexName returns the indexname
func (s *SimpleRecorder) IndexName() string { return s.indexName }

// Timeout returns the timeout
func (s *SimpleRecorder) Timeout() time.Duration { return s.timeout }
