// Copyright 2016 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fs

import (
	"os"
	"syscall"
	"testing"
)

//--------
// Statfs
//--------

func TestStatfsRoot(t *testing.T) {
	f := func(fs FSImplem, t *T) {

		buf := &syscall.Statfs_t{}
		name := fs.Root()
		if err := syscall.Statfs(name, buf); err != nil {
			t.Fatalf(
				"[%v] Statfs(%s): expected no error, got '%v'\n",
				fs, name, err)
		}

	}
	TestAllImplem(t, f)
}

func TestStatfsENOENT(t *testing.T) {
	f := func(fs FSImplem, t *T) {

		buf := &syscall.Statfs_t{}
		name := fs.Root() + "nonexisting"
		expect := syscall.ENOENT
		if err := syscall.Statfs(name, buf); err != expect {
			t.Fatalf(
				"[%v] Statfs(%s): expected '%v', got '%v'\n",
				fs, name, expect, err)
		}

	}
	TestAllImplem(t, f)
}

//func TestStatfsEACCES(t *testing.T) {
//	f := func(fs FSImplem, t *T) {
//		name := fs.Root() + "file"
//		t.WriteFile(name, []byte("some data"), os.ModeAppend)
//		if err := os.Chmod(name, 0); err != nil {
//			t.Fatalf(
//				"[%v] Chmod(%s, %q): '%v'\n",
//				fs, name, 0, err)
//		}
//
//		buf := &syscall.Statfs_t{}
//		expect := syscall.EACCES
//		if err := syscall.Statfs(name, buf); err != expect {
//			t.Fatalf(
//				"[%v] Statfs(%s): expected '%v', got '%v'\n",
//				fs, name, expect, err)
//		}
//
//	}
//	TestAllImplem(t, f)
//}

func TestStatfsENOTDIR(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		// legit file
		name := fs.Root() + "file"
		t.WriteFile(name, []byte("some data"), os.ModeAppend)

		name = name + "/notachild"
		buf := &syscall.Statfs_t{}
		expect := syscall.ENOTDIR
		if err := syscall.Statfs(name, buf); err != expect {
			t.Fatalf(
				"[%v] Statfs(%s): expected '%v', got '%v'\n",
				fs, name, expect, err)
		}
	}
	TestAllImplem(t, f)
}

//------
// Stat
//------

//---------
// ReadDir (ioutil)
//---------

//------
// Open (os)
//------

//-------
// Chmod
//-------

//-------
// Chown
//-------

//----------
// Truncate
//----------

//----------
// Readlink
//----------

//--------
// Unlink
//--------

//--------
// Remove (os)
//--------

//-------
// Mkdir
//-------

//--------
// Rename
//--------

//--------
// Access
//--------

//-------
// Write
//-------

//------
// Read
//------
