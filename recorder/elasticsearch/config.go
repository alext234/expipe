// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

package elasticsearch

import (
    "context"
    "fmt"
    "time"

    "github.com/Sirupsen/logrus"
    "github.com/arsham/expvastic/lib"
    "github.com/arsham/expvastic/recorder"
    "github.com/spf13/viper"
)

// Config holds the necessary configuration for setting up an elasticsearch reader endpoint.
type Config struct {
    name       string
    Endpoint_  string `mapstructure:"endpoint"`
    Timeout_   string `mapstructure:"timeout"`
    LogLevel_  string `mapstructure:"log_level"`
    Backoff_   int    `mapstructure:"backoff"`
    IndexName_ string `mapstructure:"index_name"`
    TypeName_  string `mapstructure:"type_name"`

    logger   logrus.FieldLogger
    interval time.Duration
    timeout  time.Duration
}

// FromViper constructs the necessary configuration for bootstrapping the elasticsearch reader
func FromViper(v *viper.Viper, name, key string) (*Config, error) {
    var (
        c         Config
        inter, to time.Duration
    )
    err := v.UnmarshalKey(key, &c)
    if err != nil {
        return nil, fmt.Errorf("decodeing config: %s", err)
    }
    if to, err = time.ParseDuration(c.Timeout_); err != nil {
        return nil, fmt.Errorf("parse timeout: %s", err)
    }
    if c.Backoff_ <= 5 {
        return nil, fmt.Errorf("back off should be at least 5: %d", c.Backoff_)
    }
    c.interval, c.timeout = inter, to

    c.logger = logrus.StandardLogger()
    if c.LogLevel_ != "" {
        c.logger = lib.GetLogger(c.LogLevel_)
    }

    if c.Endpoint_ == "" {
        return nil, fmt.Errorf("endpoint cannot be empty")
    }
    url, err := lib.SanitiseURL(c.Endpoint_)
    if err != nil {
        return nil, fmt.Errorf("invalid endpoint: %d", c.Endpoint_)
    }
    c.Endpoint_ = url

    c.name = name
    return &c, nil
}

func (c *Config) NewInstance(ctx context.Context) (recorder.DataRecorder, error) {
    return NewElasticSearch(ctx, c.logger, c.Endpoint(), c.IndexName())
}
func (c *Config) Name() string               { return c.name }
func (c *Config) IndexName() string          { return c.IndexName_ }
func (c *Config) TypeName() string           { return c.TypeName_ }
func (c *Config) Endpoint() string           { return c.Endpoint_ }
func (c *Config) RoutePath() string          { return "" }
func (c *Config) Interval() time.Duration    { return c.interval }
func (c *Config) Timeout() time.Duration     { return c.timeout }
func (c *Config) Logger() logrus.FieldLogger { return c.logger }
func (c *Config) Backoff() int               { return c.Backoff_ }