// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

// Package config contains the logic for reading the configurations from a file
// in order to bootstrap it.
package config

import (
	"strings"

	"github.com/alext234/expipe/reader"
	"github.com/alext234/expipe/recorder"

	"github.com/alext234/expipe/reader/expvar"
	"github.com/alext234/expipe/reader/self"
	"github.com/alext234/expipe/recorder/elasticsearch"
	"github.com/alext234/expipe/tools"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	selfReader            = "self"
	expvarReader          = "expvar"
	elasticsearchRecorder = "elasticsearch"
)

// routeMap looks like this:
// {
//     route1: {readers: [my_app, self], recorders: [elastic1]}
//     route2: {readers: [my_app], recorders: [elastic1, file1]}
// }
type routeMap map[string]route
type route struct {
	readers   []string
	recorders []string
}

// ConfMap holds the relation between readers and recorders.
type ConfMap struct {
	// Readers contains a map of reader names to their instantiated objects.
	Readers map[string]reader.DataReader

	// Recorders contains a map of recorder names to their instantiated objects.
	Recorders map[string]recorder.DataRecorder

	// Routes contains a map of reader names to a list of recorders.
	// map["red1"][]string{"rec1", "rec2"}: means whatever is read
	// from red1, will be shipped to rec1 and rec2.
	Routes map[string][]string
}

// Checks the application scope settings. Applies them if defined. If the log
// level is defined, it will replace a new logger with the provided one.
func checkSettingsSect(log *tools.Logger, v *viper.Viper) error {
	if v.IsSet("settings.log_level") {
		newLevel, ok := v.Get("settings.log_level").(string)
		if !ok {
			return &StructureErr{"log_level", "should be a string", nil}
		}
		*log = *tools.GetLogger(newLevel)
	}
	return nil
}

// LoadYAML loads the settings from the configuration file. It returns any
// errors returned from readers/recorders. Please refer to their documentations.
func LoadYAML(log *tools.Logger, v *viper.Viper) (*ConfMap, error) {
	var (
		readerKeys   map[string]string
		recorderKeys map[string]string
		routes       routeMap
		err          error
	)
	if len(v.AllSettings()) == 0 {
		return nil, ErrEmptyConfig
	}
	if v.IsSet("settings") {
		if err = checkSettingsSect(log, v); err != nil {
			return nil, &StructureErr{"settings", "", err}
		}
	}

	if readerKeys, err = getReaders(v); err != nil {
		return nil, errors.WithMessage(err, "readerKeys")
	}
	if recorderKeys, err = getRecorders(v); err != nil {
		return nil, errors.WithMessage(err, "recorderKeys")
	}
	if routes, err = getRoutes(v); err != nil {
		return nil, errors.WithMessage(err, "routes")
	}
	if err = checkAgainstReadRecorders(routes, readerKeys, recorderKeys); err != nil {
		return nil, errors.WithMessage(err, "checkAgainstReadRecorders")
	}
	return loadConfiguration(v, log, routes, readerKeys, recorderKeys)
}

// readers is a map of keyName:typeName
// typeName is not the recorder's type, it's the extension name, e.g. expvar.
func getReaders(v *viper.Viper) (map[string]string, error) {
	readers := make(map[string]string)
	if !v.IsSet("readers") {
		return nil, NewNotSpecifiedError("readers", "", nil)
	}
	for reader := range v.GetStringMap("readers") {
		switch rType := v.GetString("readers." + reader + ".type"); rType {
		case selfReader:
			readers[reader] = rType
		case expvarReader:
			readers[reader] = rType
		case "":
			fallthrough
		default:
			return nil, NewNotSpecifiedError(reader, "type", nil)
		}
	}
	return readers, nil
}

// recorders is a map of keyName:typeName
// typeName is not the recorder's type, it's the extension name, e.g. elasticsearch.
func getRecorders(v *viper.Viper) (map[string]string, error) {
	recorders := make(map[string]string)
	if !v.IsSet("recorders") {
		return nil, NewNotSpecifiedError("recorders", "", nil)
	}
	for recorder := range v.GetStringMap("recorders") {
		switch rType := v.GetString("recorders." + recorder + ".type"); rType {
		case elasticsearchRecorder:
			recorders[recorder] = rType
		case "":
			fallthrough
		default:
			return nil, NewNotSpecifiedError(recorder, "type", nil)
		}
	}
	return recorders, nil
}

func getRoutes(v *viper.Viper) (routeMap, error) {
	routes := make(map[string]route)
	if !v.IsSet("routes") {
		return nil, NewNotSpecifiedError("routes", "", nil)
	}
	for name := range v.GetStringMap("routes") {
		rt := route{}
		for recRedType, list := range v.GetStringMapStringSlice("routes." + name) {
			for _, target := range list {
				if strings.Contains(target, ",") {
					return nil, NewRoutersError(recRedType, "not an array or single value", nil)
				}

				if recRedType == "readers" {
					rt.readers = append(rt.readers, target)
				} else if recRedType == "recorders" {
					rt.recorders = append(rt.recorders, target)
				}
			}
			routes[name] = rt
		}

		if len(routes[name].readers) == 0 {
			return nil, NewRoutersError("readers", "is empty", nil)
		}

		if len(routes[name].recorders) == 0 {
			return nil, NewRoutersError("recorders", "is empty", nil)
		}
	}
	return routes, nil
}

// Checks all apps in routes are mentioned in the readerKeys and recorderKeys.
func checkAgainstReadRecorders(routes routeMap, readerKeys, recorderKeys map[string]string) error {
	for _, section := range routes {
		for _, reader := range section.readers {
			if !tools.StringInMapKeys(reader, readerKeys) {
				return NewRoutersError("routers", reader+" not in readers", nil)
			}
		}
		for _, recorder := range section.recorders {
			if !tools.StringInMapKeys(recorder, recorderKeys) {
				return NewRoutersError("routers", recorder+" not in recorders", nil)
			}
		}
	}
	return nil
}

func loadConfiguration(v *viper.Viper, log tools.FieldLogger, routes routeMap, readerKeys, recorderKeys map[string]string) (*ConfMap, error) {
	confMap := &ConfMap{
		Readers:   make(map[string]reader.DataReader, len(readerKeys)),
		Recorders: make(map[string]recorder.DataRecorder, len(recorderKeys)),
	}
	for name, reader := range readerKeys {
		r, err := parseReader(v, log, reader, name)
		if err != nil {
			return nil, errors.Wrap(err, "reader keys")
		}
		if !readerInRoutes(name, routes) {
			continue
		}
		confMap.Readers[name] = r
	}

	for name, recorder := range recorderKeys {
		r, err := readRecorders(v, log, recorder, name)
		if err != nil {
			return nil, errors.Wrap(err, "recorder keys")
		}
		if !recorderInRoutes(name, routes) {
			continue
		}
		confMap.Recorders[name] = r
	}
	confMap.Routes = mapReadersRecorders(routes)
	return confMap, nil
}

func readerInRoutes(name string, routes routeMap) bool {
	for _, r := range routes {
		if tools.StringInSlice(name, r.readers) {
			return true
		}
	}
	return false
}

func recorderInRoutes(name string, routes routeMap) bool {
	for _, r := range routes {
		if tools.StringInSlice(name, r.recorders) {
			return true
		}
	}
	return false
}

func parseReader(v *viper.Viper, log tools.FieldLogger, readerType, name string) (reader.DataReader, error) {
	switch readerType {
	case expvarReader:
		rc, err := expvar.NewConfig(
			expvar.WithLogger(log),
			expvar.WithViper(v, name, "readers."+name),
		)
		if err != nil {
			return nil, errors.Wrap(err, "parsing reader")
		}
		return rc.Reader()
	case selfReader:
		rc, err := self.NewConfig(
			self.WithLogger(log),
			self.WithViper(v, name, "readers."+name),
		)
		if err != nil {
			return nil, errors.Wrap(err, "parsing reader")
		}
		return rc.Reader()
	}
	return nil, NotSupportedError(readerType)
}

func readRecorders(v *viper.Viper, log tools.FieldLogger, recorderType, name string) (recorder.DataRecorder, error) {
	switch recorderType {
	case elasticsearchRecorder:
		rc, err := elasticsearch.NewConfig(
			elasticsearch.WithViper(v, name, "recorders."+name),
			elasticsearch.WithLogger(log),
		)
		if err != nil {
			return nil, errors.Wrap(err, "read-recorders loading from viper")
		}
		return rc.Recorder()
	}
	return nil, NotSupportedError(recorderType)
}

// This function returns a map of reader->recorders
// TODO: [refactor] this code
func mapReadersRecorders(routes routeMap) map[string][]string {
	// We don't know how this matrix will be, let's go dynamic! This looks ugly.
	// The whole logic should change. But it doesn't have any impact on the
	// program, it just runs once.
	readerMap := make(map[string][]string) //
	for _, route := range routes {
		// Add the readers to the map
		for _, redName := range route.readers {
			// now iterate through the recorders and add them
			for _, recName := range route.recorders {
				if _, ok := readerMap[redName]; !ok {
					readerMap[redName] = []string{recName}
				} else {
					readerMap[redName] = append(readerMap[redName], recName)
					// Shall we go another level deep??? :p I'm kidding,
					// seriously, refactor this thing Do you know why the
					// chicken crossed the road? There was a few nested eggs on
					// the other side! Okay, back to the business. BTW ask me
					// why I left these comments.
				}
			}
		}
	}

	// Let's clean up
	resultMap := make(map[string][]string)
	for redName, redsddd := range readerMap {
		checkMap := make(map[string]bool)
		for _, recName := range redsddd {
			if _, ok := checkMap[recName]; !ok {
				checkMap[recName] = true
				if _, ok := resultMap[redName]; !ok {
					resultMap[redName] = []string{recName}
				} else {
					resultMap[redName] = append(resultMap[redName], recName)
					// Remember that chicken? It's roasted now.
				}
			}
		}
	}
	return resultMap
}
