// Copyright 2020 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package tinylru

// DefaultSize is the default maximum size of an LRU cache before older items
// get automatically evicted.
const DefaultSize = 256

// LRU implements an LRU cache
type LRU = LRUG[interface{}, interface{}]
