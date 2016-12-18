// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

package expvastic

import (
	"context"
	"time"

	"github.com/Sirupsen/logrus"
)

// Engine represents an engine that receives information from readers and ships them to recorders.
// The Engine is allowed to change the index and type names at will.
// When the context times out or canceled, the engine will close the the job channels by calling the Stop method.
type Engine struct {
	ctx          context.Context // Will call Stop() when this context is canceled/timedout.
	targetReader TargetReader    // The worker that reads from an expvar provider.
	recorder     DataRecorder    // Recorder (e.g. ElasticSearch) client.
	indexName    string          // Recorder (e.g. ElasticSearch) index name.
	typeName     string          // Recorder (e.g. ElasticSearch) type name.
	interval     time.Duration
	timeout      time.Duration
	logger       logrus.FieldLogger
}

// NewEngine copies its configurations from c.
func NewEngine(ctx context.Context, c Conf) *Engine {
	cl := &Engine{
		ctx:          ctx,
		recorder:     c.Recorder,
		targetReader: c.TargetReader,
		indexName:    c.IndexName,
		typeName:     c.TypeName,
		interval:     c.Interval,
		timeout:      c.Timeout,
		logger:       c.Logger,
	}
	return cl
}

// Start begins pulling the data from TargetReader and chips them to DataRecorder.
// When the context cancels or timesout, the engine closes all job channels, causing the readers and recorders to stop.
func (c *Engine) Start() {
	resultChan := c.targetReader.ResultChan()
	ticker := time.NewTicker(c.interval)
	for {
		select {
		case <-ticker.C:
			go issueReaderJob(c.ctx, c.logger, c.targetReader, c.timeout)
		case r := <-resultChan:
			go redirectToRecorder(c.ctx, c.logger, r, c.recorder, c.timeout, c.indexName, c.typeName)
		case <-c.ctx.Done():
			c.Stop()
			return
		}
	}
}

// Stop closes the job channels
func (c *Engine) Stop() {
	close(c.targetReader.JobChan())
	close(c.recorder.PayloadChan())
	// TODO: ask the readers/recorders for their done channels and wait until they are closed.
}

func issueReaderJob(ctx context.Context, logger logrus.FieldLogger, reader TargetReader, timeout time.Duration) {
	ctx, _ = context.WithTimeout(ctx, timeout)
	timer := time.NewTimer(timeout)
	select {
	case reader.JobChan() <- ctx:
		timer.Stop()
		return
	case <-timer.C: // QUESTION: Do I need this? Or should I apply the same for recorder?
		logger.Warn("timedout before receiving the error")
	case <-ctx.Done():
		logger.Warnf("timedout before receiving the error response: %s", ctx.Err())
	}

}

func redirectToRecorder(ctx context.Context, logger logrus.FieldLogger, r *ReadJobResult, p DataRecorder, timeout time.Duration, indexName, typeName string) {
	defer r.Res.Close()
	ctx, _ = context.WithTimeout(ctx, timeout)
	errChan := make(chan error)
	payload := &RecordJob{
		Ctx:       ctx,
		Payload:   jobResultDataTypes(r.Res),
		IndexName: indexName,
		TypeName:  typeName,
		Time:      r.Time,
		Err:       errChan,
	}
	p.PayloadChan() <- payload
	select {
	case err := <-errChan:
		if err != nil {
			logger.Errorf("%s", err)
		}
	case <-ctx.Done():
		logger.Warnf("timedout before receiving the error%s", ctx.Err())
	}
}
