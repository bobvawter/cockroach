// Copyright 2018 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License. See the AUTHORS file
// for names of contributors.

package gen

import (
	"bytes"
	"io"

	"github.com/cockroachdb/cockroach/pkg/util/syncutil"
)

// mapWriter is a trivial implementation of io.WriteCloser that captures
// its output in a map. Access to the map is synchronized via a
// shared mutex.
type mapWriter struct {
	buf  bytes.Buffer
	name string
	mu   struct {
		*syncutil.Mutex
		dest map[string][]byte
	}
}

func newMapWriter(name string, mu *syncutil.Mutex, outputs map[string][]byte) io.WriteCloser {
	ret := &mapWriter{name: name}
	ret.mu.Mutex = mu
	ret.mu.dest = outputs
	return ret
}

// Write implements io.Writer.
func (w *mapWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

// Close implements io.Closer.
func (w *mapWriter) Close() error {
	w.mu.Lock()
	if w.mu.dest != nil {
		w.mu.dest[w.name] = w.buf.Bytes()
	}
	w.mu.Unlock()
	return nil
}
