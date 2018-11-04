// Copyright 2016 Arsham Shirvani <arshamshirvani@gmail.com>. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license
// License that can be found in the LICENSE file.

// Package engine can read from any endpoints that provides expvar data and
// ships them to elasticsearch. You can inspect the metrics with kibana.
//
// The Operator implements the Engine interface. It is allowed to change the
// index and type names at will. When the context times out or cancelled, the
// Engine will close and return. Use the shut down channel to signal the Engine
// to stop recording. The ctx context will create a new context based on the
// parent.
//
// Please refer to golang's expvar documentation for more information.
// Installation guides can be found on github page:
// https://github.com/alext234/expipe
//
// At the heart of this package, there is Engine. It acts like a glue between
// multiple Readers and a Recorder. Messages are transferred in a package called
// DataContainer, which is a list of DataType objects.
//
// Collected metrics
//
// This list will grow in time:
//
//   +----------------------+-------------------------+
//   |   Expipe var name    |  ElasticSearch Var Name |
//   +----------------------+-------------------------+
//   | expRecorders         | Recorders               |
//   | readJobs             | Read Jobs               |
//   | recordJobs           | Record Jobs             |
//   | datatypeObjs         | DataType Objects        |
//   +----------------------+-------------------------+
//
// Example configuration
//
// Save it somewhere (let's call it expipe.yml for now):
//
//    settings:
//        log_level: info
//
//    readers:                           # You can specify the applications you want to show the metrics
//        FirstApp:                      # service name
//            type: expvar               # the type of reader. More to come soon!
//            type_name: AppVastic       # this will be the _type in elasticsearch
//            endpoint: localhost:1234   # where the application
//            routepath: /debug/vars     # the endpoint that app provides the metrics
//            interval: 500ms            # every half a second, it will collect the metrics.
//            timeout: 3s                # in 3 seconds it gives in if the application is not responsive
//        AnotherApplication:
//            type: expvar
//            type_name: this_is_awesome
//            endpoint: localhost:1235
//            routepath: /metrics
//            timeout: 13s
//
//    recorders:                         # This section is where the data will be shipped to
//        main_elasticsearch:
//            type: elasticsearch        # the type of recorder. More to come soon!
//            endpoint: 127.0.0.1:9200
//            index_name: expipe
//            timeout: 8s
//        the_other_elasticsearch:
//            type: elasticsearch
//            endpoint: 127.0.0.1:9201
//            index_name: expipe
//            timeout: 18s
//
//    routes:                            # You can specify metrics of which application will be recorded in which target
//        route1:
//            readers:
//                - FirstApp
//            recorders:
//                - main_elasticsearch
//        route2:
//            readers:
//                - FirstApp
//                - AnotherApplication
//            recorders:
//                - main_elasticsearch
//        route3:                        # Yes, you can have multiple!
//            readers:
//                - AnotherApplication
//            recorders:
//                - main_elasticsearch
//                - the_other_elasticsearch
//
// Then run the application:
//
//    expipe -c expipe.yml
//
// You can mix and match the routes, but the engine will choose the best set-up
// to achieve your goal without duplicating the results. For instance assume
// you set the routes like this:
//
//     readers:
//         app_0: type: expvar
//         app_1: type: expvar
//         app_2: type: expvar
//         app_3: type: expvar
//         app_4: type: expvar
//         app_5: type: expvar
//         not_used_app: type: expvar # note that this one is not specified in the routes, therefore it is ignored
//     recorders:
//         elastic_0: type: elasticsearch
//         elastic_1: type: elasticsearch
//         elastic_2: type: elasticsearch
//         elastic_3: type: elasticsearch
//     routes:
//         route1:
//             readers:
//                 - app_0
//                 - app_2
//                 - app_4
//             recorders:
//                 - elastic_1
//         route2:
//             readers:
//                 - app_0
//                 - app_5
//             recorders:
//                 - elastic_2
//                 - elastic_3
//         route3:
//             readers:
//                 - app_1
//                 - app_2
//             recorders:
//                 - elastic_0
//                 - elastic_1
//
// Expipe creates three engines like so:
//
//     elastic_0 records data from app_0, app_1
//     elastic_1 records data from app_0, app_1, app_2, app_4
//     elastic_2 records data from app_0, app_5
//     elastic_3 records data from app_0, app_5
//
// You can change the numbers to your liking:
//
//     gc_types:              # These inputs will be collected into one list and zero values will be removed
//         memstats.PauseEnd
//         memstats.PauseNs
//
//     memory_bytes:           # These values will be transformed from bytes
//         StackInuse: mb      # To MB
//         memstats.Alloc: gb  # To GB
//
// To run the tests for the codes, in the root of the application run:
//   go test $(glide nv)
//
// Or for testing readers:
//
//    go test ./readers
//
// To show the coverage, use this gist:
// https://gist.github.com/alext234/f45f7e7eea7e18796bc1ed5ced9f9f4a. Then run:
//
//   gocover
//
// It will open a browser tab and show you the coverage.
//
// To run all benchmarks:
//
//    go test $(glide nv) -run=^$ -bench=.
//
// For showing the memory and cpu profiles, on each folder run:
//
//   BASENAME=$(basename $(pwd))
//   go test -run=^$ -bench=. -cpuprofile=cpu.out -benchmem -memprofile=mem.out
//   go tool pprof -pdf $BASENAME.test cpu.out > cpu.pdf && open cpu.pdf
//   go tool pprof -pdf $BASENAME.test mem.out > mem.pdf && open mem.pdf
//
// License
//
// Use of this source code is governed by the Apache 2.0 license.
// License that can be found in the LICENSE file.
package engine
