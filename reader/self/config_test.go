// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

package self_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/alext234/expipe/reader/self"
	"github.com/alext234/expipe/tools"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

func TestWithLogger(t *testing.T) {
	l := (tools.FieldLogger)(nil)
	c := new(self.Config)
	err := self.WithLogger(l)(c)
	if err == nil {
		t.Error("err = (nil); want (error)")
	}
	l = tools.DiscardLogger()
	err = self.WithLogger(l)(c)
	if err != nil {
		t.Errorf("err = (%v); want (nil)", err)
	}
	if c.Logger() != l {
		t.Errorf("c.Logger() = (%v); want (%v)", c.Logger(), l)
	}
}

type unmarshaller interface {
	UnmarshalKey(key string, rawVal interface{}) error
	AllKeys() []string
}

func TestWithViper(t *testing.T) {
	tcs := []struct {
		tcName string
		name   string
		key    string
		v      unmarshaller
	}{
		{"no name", "", "key", viper.New()},
		{"no key", "name", "", viper.New()},
		{"no viper", "name", "key", nil},
	}

	for _, tc := range tcs {
		t.Run(tc.tcName, func(t *testing.T) {
			c := new(self.Config)
			err := self.WithViper(tc.v, tc.name, tc.key)(c)
			if err == nil {
				t.Error("err = (nil); want (error)")
			}
		})
	}
}

func TestWithViperSuccess(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")

	input := bytes.NewBuffer([]byte(`
    recorders:
        recorder1:
            endpoint: http://127.0.0.1:9200
            type_name: example_type
            map_file: noway
            timeout: 10s
            interval: 1s
    `))
	v.ReadConfig(input)
	c := new(self.Config)
	err := self.WithViper(v, "recorder1", "recorders.recorder1")(c)
	if err != nil {
		t.Fatalf("err = (%v); want (nil)", err)
	}
	if c.Endpoint() != "http://127.0.0.1:9200" {
		t.Errorf("c.Endpoint() = (%s); want (http://127.0.0.1:9200)", c.Endpoint())
	}
	if c.TypeName() != "example_type" {
		t.Errorf("c.TypeName() = (%s); want (example_type)", c.TypeName())
	}
}

type badMarshaller struct{}

func (badMarshaller) UnmarshalKey(key string, rawVal interface{}) error { return errors.New("text") }
func (badMarshaller) AllKeys() []string                                 { return []string{} }

func TestWithViperBadFile(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	c := new(self.Config)
	tcs := []struct {
		name  string
		input *bytes.Buffer
	}{
		{
			name: "timeout",
			input: bytes.NewBuffer([]byte(`
    recorders:
        recorder1:
                index_name: example_index
                timeout: abc
                interval: 1s
    `)),
		},
		{
			name: "bad interval",
			input: bytes.NewBuffer([]byte(`
    recorders:
        recorder1:
                index_name: example_index
                timeout: 1s
                interval: def
    `)),
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			v.ReadConfig(tc.input)
			err := self.WithViper(v, "recorder1", "recorders.recorder1")(c)
			if err == nil {
				t.Error("err = (nil); want (error)")
			}
		})
	}

	err := self.WithViper(&badMarshaller{}, "recorder1", "recorders.recorder1")(c)
	if err == nil {
		t.Error("err = (nil); want (error)")
	}
}

func TestNewConfig(t *testing.T) {
	log := tools.DiscardLogger()
	c, err := self.NewConfig(
		self.WithLogger(log),
	)
	if err != nil {
		t.Errorf("err = (%v); want (nil)", err)
	}
	if c == nil {
		t.Error("c = (nil); want (Config)")
	}
	c, err = self.NewConfig(
		self.WithLogger(nil),
	)
	if err == nil {
		t.Error("err = (nil); want (error)")
	}
	if c != nil {
		t.Errorf("c = (%v); want (nil)", c)
	}
}

func TestConfigReader(t *testing.T) {
	log := tools.DiscardLogger()
	c, err := self.NewConfig(
		self.WithLogger(log),
	)
	c.SelfName = "name"
	c.SelfTypeName = "name"
	c.SelfEndpoint = "http://localhost"
	if err != nil {
		t.Fatalf("err = (%v); want (nil)", err)
	}
	e, err := c.Reader()
	if err == nil {
		t.Error("err = (nil); want (error)")
	}
	if e.(*self.Reader) != nil {
		t.Errorf("e.(*self.Reader): e = (%v); want (nil)", e)
	}

	c.Cinterval = time.Second
	e, err = c.Reader()
	if err != nil {
		t.Errorf("err = (%v); want (nil)", err)
	}
	if e.(*self.Reader) == nil {
		t.Error("e.(*self.Reader) = (nil); want (c = Reader)")
	}
	if e.Endpoint() == c.SelfEndpoint {
		t.Errorf("Endpoint() = (%s); want (tempServer)", e.Endpoint())
	}
}
