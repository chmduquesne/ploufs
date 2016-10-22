// Copyright 2016 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fs

import (
	"io/ioutil"
	"os"
	"syscall"
	"testing"
	"time"
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
		if !os.IsPermission(err) {
			t.Fatalf(
				"[%v] ReadDir(%s): expected an error when opening a file, got '%v'\n",
				fs, name, err)
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
		if !os.IsPermission(err) {
			t.Fatalf(
				"[%v] ReadDir(%s): expected a permission error, got '%v'\n",
				fs, name, err)
		}
	}
	TestAllImplem(t, f)
}

func TestReaddirENOENT(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		name := fs.Root()

		name = name + "/nonexisting"
		_, err := ioutil.ReadDir(name)
		if !os.IsNotExist(err) {
			t.Fatalf(
				"[%v] ReadDir(%s): expected an existence error, got '%v'\n",
				fs, name, err)
		}
	}
	TestAllImplem(t, f)
}

func TestReaddirEACCES(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		name := fs.Root() + "/dir"
		t.Mkdir(name, 0000)

		_, err := ioutil.ReadDir(name)
		if !os.IsPermission(err) {
			t.Fatalf(
				"[%v] ReadDir(%s): expected a permission error, got '%v'\n",
				fs, name, err)
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

func TestChownNoChange(t *testing.T) {
	f := func(fs FSImplem, t *T) {

		name := fs.Root()
		// Create a file
		name = name + "/file"
		t.WriteFile(name, []byte("some data"), os.ModeAppend)

		uid := os.Getuid()
		gid := os.Getgid()

		err := os.Chown(name, uid, gid)
		if err != nil {
			t.Fatalf(
				"[%v] Chown(%s): expected no error when not changing anything\n",
				fs, name)
		}
	}
	TestAllImplem(t, f)
}

//----------
// Truncate
//----------

func TestTruncateZero(t *testing.T) {
	f := func(fs FSImplem, t *T) {

		name := fs.Root()
		// Create a file
		name = name + "/file"
		t.WriteFile(name, []byte("some data"), 0700)

		sz := int64(0)
		err := os.Truncate(name, sz)
		if err != nil {
			t.Fatalf(
				"[%v] Truncate(%s): expected no error, got %v\n",
				fs, name, err)
		}

		info, _ := os.Stat(name)
		if info.Size() != sz {
			t.Fatalf(
				"[%v] After truncate(%s): expected size %v, got %v\n",
				fs, name, sz, info.Size())
		}
	}
	TestAllImplem(t, f)
}

func TestTruncateExtend(t *testing.T) {
	f := func(fs FSImplem, t *T) {

		name := fs.Root()
		// Create a file
		name = name + "/file"
		data := []byte("hello world!")
		t.WriteFile(name, data, 0700)

		sz := int64(1024)
		err := os.Truncate(name, sz)
		if err != nil {
			t.Fatalf(
				"[%v] Truncate(%s): expected no error, got %v\n",
				fs, name, err)
		}

		info, _ := os.Stat(name)
		if info.Size() != sz {
			t.Fatalf(
				"[%v] After truncate(%s): expected size %v, got %v\n",
				fs, name, sz, info.Size())
		}

		// man 2 truncate
		// If the file previously was shorter, it is extended, and the
		// extended part reads as null bytes ('\0')
		expected := make([]byte, sz)
		copy(expected, data)

		content, _ := ioutil.ReadFile(name)
		if err := t.CompareSlices(expected, content); err != nil {
			t.Fatal("[%v] After truncate(%s): %v\n", fs, name, err)
		}

	}
	TestAllImplem(t, f)
}

func TestTruncateSameSize(t *testing.T) {
	f := func(fs FSImplem, t *T) {

		name := fs.Root()
		// Create a file
		name = name + "/file"
		data := []byte("some data")
		t.WriteFile(name, data, 0700)

		sz := int64(len(data))
		err := os.Truncate(name, sz)
		if err != nil {
			t.Fatalf(
				"[%v] Truncate(%s): expected no error, got %v\n",
				fs, name, err)
		}

		info, _ := os.Stat(name)
		if info.Size() != sz {
			t.Fatalf(
				"[%v] After truncate(%s): expected size %v, got %v\n",
				fs, name, sz, info.Size())
		}
	}
	TestAllImplem(t, f)
}

func TestTruncateOneByte(t *testing.T) {
	f := func(fs FSImplem, t *T) {

		name := fs.Root()
		// Create a file
		name = name + "/file"
		data := []byte("some data")
		t.WriteFile(name, data, 0700)

		sz := int64(len(data) - 1)
		err := os.Truncate(name, sz)
		if err != nil {
			t.Fatalf(
				"[%v] Truncate(%s): expected no error, got %v\n",
				fs, name, err)
		}

		info, _ := os.Stat(name)
		if info.Size() != sz {
			t.Fatalf(
				"[%v] After truncate(%s): expected size %v, got %v\n",
				fs, name, sz, info.Size())
		}
	}
	TestAllImplem(t, f)
}

func TestTruncateModTime(t *testing.T) {
	f := func(fs FSImplem, t *T) {

		name := fs.Root()
		// Create a file
		name = name + "/file"
		data := []byte("hello world!")
		t.WriteFile(name, data, 0700)

		info, _ := os.Stat(name)
		modTime := info.ModTime()

		// Sleeping a bit to leave a chance to modtime to change
		time.Sleep(time.Second / 100)
		os.Truncate(name, 0)

		info, _ = os.Stat(name)
		if info.ModTime().Equal(modTime) {
			t.Fatalf(
				"[%v] After truncate(%s): expected different modtime",
				fs, name)
		}
	}
	TestAllImplem(t, f)
}

func TestTruncateEACCES(t *testing.T) {
	f := func(fs FSImplem, t *T) {

		name := fs.Root()
		// Create a file
		name = name + "/file"
		data := []byte("hello world!")
		t.WriteFile(name, data, os.ModeAppend)

		err := os.Truncate(name, 0)
		if !os.IsPermission(err) {
			t.Fatalf(
				"[%v] Truncate(%s): expected permission error, got '%v'",
				fs, name, err)
		}
	}
	TestAllImplem(t, f)
}

func TestTruncateEISDIR(t *testing.T) {
	f := func(fs FSImplem, t *T) {

		name := fs.Root()
		err := os.Truncate(name, 0).(*os.PathError)
		if err.Err != syscall.EISDIR {
			t.Fatalf(
				"[%v] Truncate(%s): expected permission error, got '%v'",
				fs, name, err.Err)
		}
	}
	TestAllImplem(t, f)
}

func TestTruncateENOENT(t *testing.T) {
	f := func(fs FSImplem, t *T) {

		name := fs.Root() + "/doesnotexist"
		err := os.Truncate(name, 0)
		if !os.IsNotExist(err) {
			t.Fatalf(
				"[%v] Truncate(%s): expected not exist error, got '%v'",
				fs, name, err)
		}
	}
	TestAllImplem(t, f)
}

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

func TestRenameFile(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		root := fs.Root()

		old := root + "/old"
		t.WriteFile(old, []byte("hello world!"), 0700)
		new := root + "/new"

		if err := os.Rename(old, new); err != nil {
			t.Fatalf(
				"[%v] Rename('%s', '%s'): expected no error, got %v",
				fs, old, new, err)
		}

	}
	TestAllImplem(t, f)
}

func TestRenameFileCheckContent(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		root := fs.Root()

		data := []byte("some data")
		old := root + "/old"
		t.WriteFile(old, data, 0700)
		new := root + "/new"

		os.Rename(old, new)

		content, err := ioutil.ReadFile(new)
		if err != nil {
			t.Fatalf(
				"[%v] ReadFile('%s'): expected no error, got %v",
				fs, new, err)
		}
		if err := t.CompareSlices(data, content); err != nil {
			t.Fatalf(
				"[%v] ReadFile('%s'): %v",
				fs, new, err)
		}
	}
	TestAllImplem(t, f)
}

func TestRenameDirectory(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		root := fs.Root()

		old := root + "/old"
		t.Mkdir(old, 0700)
		new := root + "/new"

		os.Rename(old, new)

		info, err := os.Stat(new)
		if err != nil {
			t.Fatalf(
				"[%v] Stat('%s'): expected no error, got %v",
				fs, new, err)
		}
		if info != nil && !info.IsDir() {
			t.Fatalf(
				"[%v] Stat('%s'): expected a directory",
				fs, new)
		}
	}
	TestAllImplem(t, f)
}

func TestRenameSymlink(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		root := fs.Root()

		old := root + "/old"
		os.Symlink(root, old)
		new := root + "/new"

		os.Rename(old, new)

		info, err := os.Lstat(new)
		if err != nil {
			t.Fatalf(
				"[%v] Lstat('%s'): expected no error, got %v",
				fs, new, err)
		}
		if info != nil && (info.Mode()&os.ModeSymlink != os.ModeSymlink) {
			t.Fatalf(
				"[%v] Lstat('%s'): expected a symlink",
				fs, new)
		}
		target, err := os.Readlink(new)
		if err != nil {
			t.Fatalf(
				"[%v] Readlink('%s'): expected no error, got %v",
				fs, new, err)
		}
		if target != root {
			t.Fatalf(
				"[%v] Readlink('%s'): expected symlink target to be %v",
				fs, new, root)
		}
	}
	TestAllImplem(t, f)
}

func TestRenameDirectoryWithChildren(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		root := fs.Root()

		old := root + "/old"
		t.Mkdir(old, 0700)
		t.Mkdir(old+"/foo", 0700)
		t.Mkdir(old+"/foo/bar", 0700)
		t.WriteFile(old+"/foo/bar/baz", []byte("some data"), 0700)
		new := root + "/new"

		os.Rename(old, new)

		for _, name := range []string{new, new + "/foo", new + "/foo/bar", new + "/foo/bar/baz"} {
			_, err := os.Stat(name)
			if err != nil {
				t.Fatalf(
					"[%v] Stat('%s'): expected no error, got %v",
					fs, name, err)
			}
		}
	}
	TestAllImplem(t, f)
}

func TestRenameDeletesOld(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		root := fs.Root()

		old := root + "/old"
		t.WriteFile(old, []byte("some data"), 0700)
		new := root + "/new"

		os.Rename(old, new)

		_, err := os.Stat(old)
		if !os.IsNotExist(err) {
			t.Fatalf(
				"[%v] Stat('%s'): expected ENOENT, got %v",
				fs, old, err)
		}
	}
	TestAllImplem(t, f)
}

func TestRenameFileToExistingFile(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		root := fs.Root()

		old := root + "/old"
		t.WriteFile(old, []byte("some data"), 0700)
		new := root + "/new"
		t.WriteFile(old, []byte("data exists"), 0700)

		err := os.Rename(old, new)

		if err != nil {
			t.Fatalf(
				"[%v] Rename('%s', '%s'): expected no error, got %v",
				fs, old, new, err)
		}
	}
	TestAllImplem(t, f)
}

func TestRenameSymlinkToExistingSymlink(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		root := fs.Root()

		targetOld := root + "/targetOld"
		t.WriteFile(targetOld, []byte("data targetOld"), 0700)
		old := root + "/old"
		os.Symlink(targetOld, old)
		targetNew := root + "/targetNew"
		t.WriteFile(targetNew, []byte("data targetNew"), 0700)
		new := root + "/new"
		os.Symlink(targetNew, new)

		err := os.Rename(old, new)

		if err != nil {
			t.Fatalf(
				"[%v] Rename('%s', '%s'): expected no error, got %v",
				fs, old, new, err)
		}
	}
	TestAllImplem(t, f)
}

func TestRenameFileToExistingSymlink(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		root := fs.Root()

		old := root + "/old"
		t.WriteFile(old, []byte("some data"), 0700)
		// new is a symlink pointing to /target
		target := root + "/target"
		t.WriteFile(target, []byte(""), 0700)
		new := root + "/new"
		os.Symlink(target, new)

		err := os.Rename(old, new)

		if err != nil {
			t.Fatalf(
				"[%v] Rename('%s', '%s'): expected no error, got %v",
				fs, old, new, err)
		}
	}
	TestAllImplem(t, f)
}

func TestRenameDirToExistingDir(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		root := fs.Root()

		old := root + "/old"
		t.Mkdir(old, 0700)
		new := root + "/new"
		os.Mkdir(new, 0700)

		err := os.Rename(old, new)

		if err != nil {
			t.Fatalf(
				"[%v] Rename('%s', '%s'): expected no error, got %v",
				fs, old, new, err)
		}
	}
	TestAllImplem(t, f)
}

func TestRenameFileEISDIR(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		root := fs.Root()

		old := root + "/old"
		t.WriteFile(old, []byte("some data"), 0700)
		new := root + "/new"
		os.Mkdir(new, 0700)

		err := os.Rename(old, new).(*os.LinkError)

		if err.Err != syscall.EISDIR {
			t.Fatalf(
				"[%v] Rename('%s', '%s'): expected '%v', got '%v'",
				fs, old, new, syscall.EISDIR, err.Err)
		}
	}
	TestAllImplem(t, f)
}

func TestRenameSymlinkEISDIR(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		root := fs.Root()

		old := root + "/old"
		os.Symlink(".", old)
		new := root + "/new"
		os.Mkdir(new, 0700)

		err := os.Rename(old, new).(*os.LinkError)

		if err.Err != syscall.EISDIR {
			t.Fatalf(
				"[%v] Rename('%s', '%s'): expected '%v', got '%v'",
				fs, old, new, syscall.EISDIR, err.Err)
		}
	}
	TestAllImplem(t, f)
}

//--------
// Access
//--------

//-------
// Write
//-------

func TestWriteFile(t *testing.T) {
	f := func(fs FSImplem, t *T) {
		root := fs.Root()

		file := root + "/file"

		f, err := os.OpenFile(file, os.O_CREATE, 0700)
		if err != nil {
			t.Fatalf(
				"[%v] Open('%s'): expected no error, got %v",
				fs, file, err)
		}
		f.Close()
	}
	TestAllImplem(t, f)
}

//------
// Read
//------
