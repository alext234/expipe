// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

package expvastic

import (
	"runtime"
	"time"

	"github.com/arsham/expvastic/datatype"
	"github.com/arsham/expvastic/reader"
	"github.com/arsham/expvastic/recorder"
	"github.com/arsham/expvastic/token"
)

// This file contains the operation section of the engine and its event loop.

// Start begins pulling the data from DataReaders and chips them to the DataRecorder.
// When the context is cancelled or timed out, the engine abandons its operations.
func (e *Engine) Start() {
	e.log.Infof("starting with %d readers", len(e.readers))
	e.shutdown = make(chan struct{})

	go func() {
		for {
			numGoroutines.Set(int64(runtime.NumGoroutine()))
			time.Sleep(50 * time.Millisecond)
		}
	}()

	e.redmu.RLock()
	readers := e.readers
	e.redmu.RUnlock()
	for _, red := range readers {
		e.wg.Add(1)
		go e.readerEventLoop(red)
	}

	e.wg.Wait()
}

// readerEventLoop starts readers event loop. It handles the recordings
func (e *Engine) readerEventLoop(red reader.DataReader) {
	expReaders.Add(1)
	ticker := time.NewTicker(red.Interval())
	e.log.Debugf("started reader: %s", red.Name())
	remove := make(chan string)
LOOP:
	for {
		select {
		case <-ticker.C:
			// [1] job's life cycle starts here...
			e.log.Debugf("issuing job to: %s", red.Name())
			waitingReadJobs.Add(1)
			go e.issueReaderJob(red, remove)

		case job := <-e.readerJobs:
			// note that the job is not necessarily from current reader as they
			// all share the same channel (see below).
			// However it's ok to ship to the recorder as they are ship their
			// results to the same recorder.
			waitingRecordJobs.Add(1)
			go e.shipToRecorder(job)

		case name := <-remove:
			// this is why we need the name
			delete(e.readers, name)
			break LOOP

		case <-e.shutdown:
			e.log.Debug("shutting down the engine")
			break LOOP

		case <-e.ctx.Done():
			e.log.Debug(contextCanceled)
			break LOOP
		}
	}

	e.wg.Done()
}

func (e *Engine) issueReaderJob(red reader.DataReader, remove chan string) {
	defer waitingReadJobs.Add(-1)
	readJobs.Add(1)

	select {
	case <-e.shutdown:
		return //the engine has been already shut down
	default:
	}

	// to make sure the reader is behaving.
	timeout := red.Timeout() + time.Duration(10*time.Second)
	timer := time.NewTimer(timeout)
	done := make(chan struct{})
	job := token.New(e.ctx)

	go func() {
		res, err := red.Read(job)
		if err != nil {
			e.log.WithField("ID", job.ID()).WithField("name", red.Name()).Error(err)
			if err == reader.ErrBackoffExceeded {
				remove <- red.Name()
			}
			return
		}
		e.readerJobs <- res
		close(done)
	}()

	select {
	case <-done:
		// job was sent, we are done here.
		if !timer.Stop() {
			<-timer.C
		}
		return

	case <-timer.C:
		erroredJobs.Add(1)
		e.log.Warn("time out before job was read")

	case <-e.ctx.Done():
		if !timer.Stop() {
			<-timer.C
		}
		erroredJobs.Add(1)
		e.log.Warn("main context closed before job was read", e.ctx.Err().Error())
	}
}

func (e *Engine) shipToRecorder(result *reader.Result) {
	defer waitingRecordJobs.Add(-1)
	res := make([]byte, len(result.Content))
	copy(res, result.Content)
	payload := datatype.JobResultDataTypes(res, result.Mapper.Copy())
	if payload.Error() != nil {
		e.log.Warnf("error in payload: %s", payload.Error())
		return
	}
	recordJobs.Add(1)
	timeout := e.recorder.Timeout() + time.Duration(10*time.Second)
	timer := time.NewTimer(timeout)
	recPayload := &recorder.Job{
		ID:        result.ID,
		Payload:   payload,
		IndexName: e.recorder.IndexName(),
		TypeName:  result.TypeName,
		Time:      result.Time,
	}

	done := make(chan struct{})
	go func() {
		// sending payload
		err := e.recorder.Record(e.ctx, recPayload)
		if err != nil {
			e.log.WithField("ID", result.ID).WithField("name", e.recorder.Name()).Error(err)
			if err == reader.ErrBackoffExceeded {
				close(e.shutdown)
			}
		}
		close(done)
	}()

	select {
	case <-done:
		// [4] job was sent
		if !timer.Stop() {
			<-timer.C
		}
		e.log.WithField("ID", result.ID).Debug("payload has been delivered")

	case <-timer.C:
		e.log.Warn("timed-out before receiving the error")

	case <-e.ctx.Done():
		e.log.WithField("ID", result.ID).Warn("main context was closed before receiving the error response", e.ctx.Err().Error())
		if !timer.Stop() {
			<-timer.C
		}
	}
}
