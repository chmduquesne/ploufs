// Copyright 2016 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fs

import (
	"syscall"
	"testing"
)

func TestStatfsRoot(t *testing.T) {
	f := func(fs FSImplem, t *T) {

		buf := &syscall.Statfs_t{}
		name := fs.Root()
		if err := syscall.Statfs(name, buf); err != nil {
			t.Fatalf(
				"When running Statfs(%s): expected no error, got '%v'\n",
				name, err)
		}

	}
	TestAllImplem(t, f)
}
