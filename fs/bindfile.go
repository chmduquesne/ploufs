// copyright 2016 Christophe-Marie Duquesne

package fs

import (
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

// BindFile delegates all operations back to an underlying os.File.
func NewBindFile(f *os.File) nodefs.File {
	return &BindFile{
		File:   nodefs.NewDefaultFile(),
		OSFile: f,
	}
}

type BindFile struct {
	nodefs.File

	OSFile *os.File
	// os.File is not threadsafe. Although fd themselves are
	// constant during the lifetime of an open file, the OS may
	// reuse the fd number after it is closed. When open races
	// with another close, they may lead to confusion as which
	// file gets written in the end.
	lock sync.Mutex
}

func (f *BindFile) SetInode(n *nodefs.Inode) {}

func (f *BindFile) String() string {
	return fmt.Sprintf("BindFile(%s)", f.OSFile.Name())
}

func (f *BindFile) InnerFile() nodefs.File {
	return nil
}

func (f *BindFile) Read(buf []byte, off int64) (res fuse.ReadResult, code fuse.Status) {
	f.lock.Lock()
	n, err := f.OSFile.ReadAt(buf, off)
	if err == io.EOF {
		err = nil
	}
	r := fuse.ReadResultData(buf[:n])
	f.lock.Unlock()
	return r, fuse.ToStatus(err)
}

func (f *BindFile) Write(data []byte, off int64) (uint32, fuse.Status) {
	f.lock.Lock()
	n, err := f.OSFile.WriteAt(data, off)
	f.lock.Unlock()
	return uint32(n), fuse.ToStatus(err)
}

func (f *BindFile) Release() {
	f.lock.Lock()
	f.OSFile.Close()
	f.lock.Unlock()
}

func (f *BindFile) Flush() fuse.Status {
	f.lock.Lock()

	// Since Flush() may be called for each dup'd fd, we don't
	// want to really close the file, we just want to flush. This
	// is achieved by closing a dup'd fd.
	newFd, err := syscall.Dup(int(f.OSFile.Fd()))
	f.lock.Unlock()

	if err != nil {
		return fuse.ToStatus(err)
	}
	err = syscall.Close(newFd)
	return fuse.ToStatus(err)
}

func (f *BindFile) Fsync(flags int) (code fuse.Status) {
	f.lock.Lock()
	r := fuse.ToStatus(syscall.Fsync(int(f.OSFile.Fd())))
	f.lock.Unlock()

	return r
}

func (f *BindFile) Allocate(off uint64, sz uint64, mode uint32) fuse.Status {
	f.lock.Lock()
	err := syscall.Fallocate(int(f.OSFile.Fd()), mode, int64(off), int64(sz))
	f.lock.Unlock()
	if err != nil {
		return fuse.ToStatus(err)
	}
	return fuse.OK
}

// Utimens - file handle based version of loopbackFileSystem.Utimens()
func (f *BindFile) Utimens(a *time.Time, m *time.Time) fuse.Status {
	futimens := func(fd int, times *[2]syscall.Timespec) (err error) {
		_, _, e1 := syscall.Syscall6(
			syscall.SYS_UTIMENSAT, uintptr(fd), 0,
			uintptr(unsafe.Pointer(times)), uintptr(0), 0, 0)
		if e1 != 0 {
			err = syscall.Errno(e1)
		}
		return
	}
	_UTIME_OMIT := int64(((1 << 30) - 2))
	var ts [2]syscall.Timespec

	if a == nil {
		ts[0].Nsec = _UTIME_OMIT
	} else {
		ts[0] = syscall.NsecToTimespec(a.UnixNano())
		ts[0].Nsec = 0
	}

	if m == nil {
		ts[1].Nsec = _UTIME_OMIT
	} else {
		ts[1] = syscall.NsecToTimespec(a.UnixNano())
		ts[1].Nsec = 0
	}

	f.lock.Lock()
	err := futimens(int(f.OSFile.Fd()), &ts)
	f.lock.Unlock()
	return fuse.ToStatus(err)
}
