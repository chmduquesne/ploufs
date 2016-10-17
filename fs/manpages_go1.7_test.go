// +build go1.7

package fs

import (
	"os"
	"os/user"
	"strconv"
	"testing"
)

//-------
// Chown
//-------

func TestChownChangeGidFile(t *testing.T) {
	f := func(fs FSImplem, t *T) {

		name := fs.Root()
		// Create a file
		name = name + "/file"
		t.WriteFile(name, []byte("some data"), os.ModeAppend)

		// Get the user running the test
		u, err := user.Current()
		if err != nil {
			t.Skip("Could not get user\n")
		}
		// Get all the groups the user belong to
		groupids, err := u.GroupIds()
		if err != nil {
			t.Skip("Could not get groups of the user\n")
		}
		// Find a group that is not the current gid
		currGid := os.Getgid()
		otherGid := 0
		for _, g := range groupids {
			gid, _ := strconv.Atoi(g)
			if gid != currGid {
				otherGid = gid
				group, _ := user.LookupGroupId(g)
				t.Logf("[%v] Chown(%s): found a different group %s\n",
					fs, name, group.Name)
				break
			}
		}
		uid := os.Getuid()
		gid := otherGid

		if otherGid != 0 {
			err = os.Chown(name, uid, gid)
			if err != nil {
				t.Fatalf(
					"[%v] Chown(%s): expected no error, got %v\n",
					fs, name, err)
			}
		}
	}
	TestAllImplem(t, f)
}

func TestChownELOOP(t *testing.T) {
	f := func(fs FSImplem, t *T) {

		root := fs.Root()
		// Create a regular file
		t.WriteFile(root+"/symlink1", []byte("some data"), os.ModeAppend)
		os.Symlink(root+"/symlink1", root+"/symlink2")
		os.Remove(root + "/symlink1")
		os.Symlink(root+"/symlink2", root+"/symlink1")
		name := root + "/symlink1"

		// Get the user running the test
		u, err := user.Current()
		if err != nil {
			t.Skip("Could not get user\n")
		}
		// Get all the groups the user belong to
		groupids, err := u.GroupIds()
		if err != nil {
			t.Skip("Could not get groups of the user\n")
		}
		// Find a group that is not the current gid
		currGid := os.Getgid()
		otherGid := 0
		for _, g := range groupids {
			gid, _ := strconv.Atoi(g)
			if gid != currGid {
				otherGid = gid
				group, _ := user.LookupGroupId(g)
				t.Logf("[%v] Chown(%s): found a different group %s\n",
					fs, name, group.Name)
				break
			}
		}
		uid := os.Getuid()
		gid := otherGid

		err = os.Chown(name, uid, gid)
		if err == nil {
			t.Fatalf(
				"[%v] Chown(%s): expected an error\n",
				fs, name)
		}
	}
	TestAllImplem(t, f)
}
