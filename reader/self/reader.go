// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

// Package self contains codes for recording expvastic's own metrics.
// Here is a list of currently collected metrics:
//
//      ElasticSearch Var Name    | expvastic var name
//      ----------------------------------------------
//      Recorders                 | expRecorders
//      Readers                   | expReaders
//      Read Jobs                 | readJobs
//      Record Jobs               | recordJobs
//      Errored Jobs              | erroredJobs
//      Records Distributed       | recordsDistributed
//      DataType Objects          | datatypeObjs
//      DataType Objects Errors   | datatypeErrs
//      Unidentified JSON Count   | unidentifiedJSON
//      StringType Count          | stringTypeCount
//      FloatType Count           | floatTypeCount
//      GCListType Count          | gcListTypeCount
//      ByteType Count            | byteTypeCount
//      Expvar Reads              | expvarReads
//      ElasticSearch Records     | elasticsearchRecords
package self

import (
	"context"
	"net"
	// to expose the metrics
	_ "expvar"
	"fmt"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/arsham/expvastic/datatype"
	"github.com/arsham/expvastic/reader"
)

// Reader contains communication channels with a worker that exposes expvar information.
// It implements DataReader interface.
type Reader struct {
	name       string
	typeName   string
	logger     logrus.FieldLogger
	mapper     datatype.Mapper
	jobChan    chan context.Context
	resultChan chan *reader.ReadJobResult
	interval   time.Duration
	url        string
}

// NewSelfReader exposes expvastic's own metrics.
func NewSelfReader(
	logger logrus.FieldLogger,
	mapper datatype.Mapper,
	jobChan chan context.Context,
	resultChan chan *reader.ReadJobResult,
	name,
	typeName string,
	interval time.Duration,
) (*Reader, error) {
	l, _ := net.Listen("tcp", ":0")
	l.Close()
	go http.ListenAndServe(l.Addr().String(), nil)
	addr := "http://" + l.Addr().String() + "/debug/vars"
	logger.Debugf("running self expvar on %s", addr)
	logger = logger.WithField("engine", "expvastic")
	w := &Reader{
		name:       name,
		typeName:   typeName,
		mapper:     mapper,
		jobChan:    jobChan,
		resultChan: resultChan,
		logger:     logger,
		interval:   interval,
		url:        addr,
	}
	return w, nil
}

// Start begins reading from the target in its own goroutine.
// It will issue a goroutine on each job request.
// It will close the done channel when the job channel is closed.
func (r *Reader) Start(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	r.logger.Debug("starting")
	go func() {
	LOOP:
		for {
			select {
			case job := <-r.jobChan:
				go r.readMetrics(job)
			case <-ctx.Done():
				break LOOP
			}
		}
		close(done)
	}()
	return done
}

// Name shows the name identifier for this reader
func (r *Reader) Name() string { return r.name }

// TypeName shows the typeName the recorder should record as
func (r *Reader) TypeName() string { return r.typeName }

// Mapper returns the mapper object
func (r *Reader) Mapper() datatype.Mapper { return r.mapper }

// Interval returns the interval
func (r *Reader) Interval() time.Duration { return r.interval }

// Timeout returns the timeout
func (r *Reader) Timeout() time.Duration { return 0 }

// JobChan returns the job channel.
func (r *Reader) JobChan() chan context.Context { return r.jobChan }

// ResultChan returns the result channel.
func (r *Reader) ResultChan() chan *reader.ReadJobResult { return r.resultChan }

// will send an error back to the engine if it can't read from metrics provider
func (r *Reader) readMetrics(job context.Context) {

	resp, err := http.Get(r.url)
	if err != nil {
		r.logger.WithField("reader", "self").Debugf("%s: error making request: %v", r.name, err)
		res := &reader.ReadJobResult{
			Time: time.Now(),
			Res:  nil,
			Err:  fmt.Errorf("making request to metrics provider: %s", err),
		}
		r.resultChan <- res
		return
	}

	res := &reader.ReadJobResult{
		Time:     time.Now(), // It is sensible to record the time now
		Res:      resp.Body,
		TypeName: r.TypeName(),
	}
	r.resultChan <- res
}