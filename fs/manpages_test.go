// Copyright 2016 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fs

import (
	"io/ioutil"
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

func TestReaddirRoot(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		// legit file
		name := fs.Root()

		entries, err := ioutil.ReadDir(name)
		if err != nil {
			t.Fatalf(
				"[%v] ReadDir(%s): expected no error, got '%v'\n",
				fs, name, err)
		}

		if len(entries) != 0 {
			t.Fatalf(
				"[%v] ReadDir(%s): expected an empty dir\n",
				fs, name)
		}
	}
	TestAllImplem(t, f)
}

func TestReaddirRegularFile(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		name := fs.Root()

		// file
		file := name + "/file"
		t.WriteFile(file, []byte("some data"), os.ModeAppend)

		entries, err := ioutil.ReadDir(name)
		if err != nil {
			t.Fatalf(
				"[%v] ReadDir(%s): expected no error, got '%v'\n",
				fs, name, err)
		}
		if len(entries) != 1 {
			t.Fatalf(
				"[%v] ReadDir(%s): expected one entry\n",
				fs, name)
		}
		if entries[0].Name() != "file" {
			t.Fatalf(
				"[%v] ReadDir(%s): expected file named %s, got %s\n",
				fs, name, "file", entries[0].Name())
		}
		if !entries[0].Mode().IsRegular() {
			t.Fatalf(
				"[%v] ReadDir(%s): expected regular file",
				fs, name)
		}
	}
	TestAllImplem(t, f)
}

func TestReaddirDir(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		name := fs.Root()

		// dir
		dir := name + "/dir"
		t.Mkdir(dir, 0700)

		entries, err := ioutil.ReadDir(name)
		if err != nil {
			t.Fatalf(
				"[%v] ReadDir(%s): expected no error, got '%v'\n",
				fs, name, err)
		}
		if len(entries) != 1 {
			t.Fatalf(
				"[%v] ReadDir(%s): expected one entry\n",
				fs, name)
		}
		if entries[0].Name() != "dir" {
			t.Fatalf(
				"[%v] ReadDir(%s): expected file named %s, got %s\n",
				fs, name, "dir", entries[0].Name())
		}
		if !entries[0].Mode().IsDir() {
			t.Fatalf(
				"[%v] ReadDir(%s): expected dir",
				fs, name)
		}
	}
	TestAllImplem(t, f)
}

func TestReaddirSymlink(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		name := fs.Root()

		// symlink
		symlink := name + "/symlink"
		os.Symlink(name, symlink)

		entries, err := ioutil.ReadDir(name)
		if err != nil {
			t.Fatalf(
				"[%v] ReadDir(%s): expected no error, got '%v'\n",
				fs, name, err)
		}
		if len(entries) != 1 {
			t.Fatalf(
				"[%v] ReadDir(%s): expected one entry\n",
				fs, name)
		}
		if entries[0].Name() != "symlink" {
			t.Fatalf(
				"[%v] ReadDir(%s): expected file named %s, got %s\n",
				fs, name, "symlink", entries[0].Name())
		}
		if entries[0].Mode()|os.ModeSymlink == 0 {
			t.Fatalf(
				"[%v] ReadDir(%s): expected symlink",
				fs, name)
		}
	}
	TestAllImplem(t, f)
}

func TestReaddirNotADir(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		name := fs.Root()

		// Create a file
		name = name + "/file"
		t.WriteFile(name, []byte("some data"), os.ModeAppend)

		_, err := ioutil.ReadDir(name)
		if err == nil {
			t.Fatalf(
				"[%v] ReadDir(%s): expected an error when opening a file\n",
				fs, name)
		}
	}
	TestAllImplem(t, f)
}

func TestReaddirOnSymlinkToDir(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		name := fs.Root()

		// Create a symlink
		os.Symlink(name, name+"/symlink")
		name = name + "/symlink"

		_, err := ioutil.ReadDir(name)
		if err != nil {
			t.Fatalf(
				"[%v] ReadDir(%s): expected no error with a symlink to a dir\n",
				fs, name)
		}
	}
	TestAllImplem(t, f)
}

func TestReaddirOnSymlinkToFile(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		name := fs.Root()

		// Create a file
		t.WriteFile(name+"/file", []byte("some data"), os.ModeAppend)
		// Create a symlink to that file
		os.Symlink(name+"/file", name+"/symlink")

		name = name + "/symlink"
		_, err := ioutil.ReadDir(name)
		if err == nil {
			t.Fatalf(
				"[%v] ReadDir(%s): expected an error with a symlink to a file\n",
				fs, name)
		}
	}
	TestAllImplem(t, f)
}

func TestReaddirENOENT(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		name := fs.Root()

		name = name + "/nonexisting"
		_, err := ioutil.ReadDir(name)
		if err == nil {
			t.Fatalf(
				"[%v] ReadDir(%s): expected an error with a non existing dir\n",
				fs, name)
		}
	}
	TestAllImplem(t, f)
}

func TestReaddirEACCES(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		name := fs.Root() + "/dir"
		t.Mkdir(name, 0000)

		_, err := ioutil.ReadDir(name)
		if err == nil {
			t.Fatalf(
				"[%v] ReadDir(%s): expected an error with a chmod 0000 dir\n",
				fs, name)
		}
	}
	TestAllImplem(t, f)
}

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
