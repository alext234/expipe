// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

package testing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alext234/expipe/reader"
	"github.com/alext234/expipe/tools/token"
	"github.com/pkg/errors"
)

// readerReceivesJob is a test helper to test the reader can receive jobs.
func readerReceivesJob(t testing.TB, cons Constructor) {
	ctx := context.Background()
	cons.SetName("the name")
	cons.SetTypeName("my type")
	cons.SetEndpoint(cons.TestServer().URL)
	cons.SetInterval(time.Hour)
	cons.SetTimeout(time.Hour)
	red, err := cons.Object()
	if errors.Cause(err) != nil {
		t.Errorf("err = (%v); want (nil)", err)
	}
	err = red.Ping()
	if errors.Cause(err) != nil {
		t.Errorf("err = (%v); want (nil)", err)
	}
	result, err := red.Read(token.New(ctx))
	if errors.Cause(err) != nil {
		t.Errorf("err = (%v); want (nil)", err)
	}
	if result == nil {
		t.Fatal("results = (nil); want (values)")
	}
	if result.ID == (token.ID{}) {
		t.Error("result.ID = (nil); want (token.ID)")
	}
	if result.TypeName == "" {
		t.Error("result.TypeName is (empty); want (TypeName)")
	}
	if result.Content == nil {
		t.Error("result.Content = (nil); want (Content)")
	}
	if result.Mapper == nil {
		t.Error("result.Mapper = (nil); want (Mapper)")
	}
}

// readerReturnsSameID is a test helper to test the reader returns the same ID
// in the response.
func readerReturnsSameID(t testing.TB, cons Constructor) {
	ctx := context.Background()
	job := token.New(ctx)
	cons.SetName("the name")
	cons.SetTypeName("my type")
	cons.SetEndpoint(cons.TestServer().URL)
	cons.SetInterval(time.Hour)
	cons.SetTimeout(time.Hour)

	red, err := cons.Object()
	if errors.Cause(err) != nil {
		t.Errorf("err = (%#v); want (nil)", err)
	}
	err = red.Ping()
	if errors.Cause(err) != nil {
		t.Errorf("err = (%#v); want (nil)", err)
	}
	result, err := red.Read(job)
	if errors.Cause(err) != nil {
		t.Errorf("err = (%#v); want (nil)", err)
	}
	if result == nil {
		t.Fatal("result = (nil); want (values)")
	}
	if result.ID != job.ID() {
		t.Errorf("result.ID = (%v); want (%v)", result.ID, job.ID())
	}
}

func jasonMarshallableCheck(t testing.TB, cons Constructor) {
	var payload string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(payload))
	}))
	defer ts.Close()

	ctx := context.Background()
	job := token.New(ctx)
	cons.SetName("the name")
	cons.SetTypeName("my type")
	cons.SetEndpoint(ts.URL)
	cons.SetInterval(time.Hour)
	cons.SetTimeout(time.Hour)
	red, err := cons.Object()
	if errors.Cause(err) != nil {
		t.Errorf("err = (%#v); want (nil)", err)
	}
	err = red.Ping()
	if errors.Cause(err) != nil {
		t.Fatalf("err = (%#v); want (nil)", err)
	}

	payload = `{"bb":1`
	result, err := red.Read(job)
	if errors.Cause(err) != reader.ErrInvalidJSON {
		t.Errorf("err = (%#v); want (reader.ErrInvalidJSON)", err)
	}
	if result != nil {
		t.Errorf("result = (%v); want (nil)", string(result.Content))
	}

	payload = `{"bb":1}`
	result, err = red.Read(job)
	if err != nil {
		t.Errorf("err = (%#v); want (nil)", err)
	}
	if result == nil {
		t.Error("result = (nil); want (reader.Result)")
	}
}
