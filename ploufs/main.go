package main

import (
	"fmt"
	"os"

	"github.com/chmduquesne/ploufs/fs"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("Usage: %s <orig> <mnt>\n\n", os.Args[0])

		fmt.Printf("Environment variables:\n")
		fmt.Printf("  ENABLE_LINKS:   if not empty, enable hard link support\n")
		fmt.Printf("  DEBUG:          if not empty, enable debugging\n")
		fmt.Printf("  MOUNT_OPTIONS:  comma separated options from man 8 mount.fuse\n")
		os.Exit(1)
	}
	fs.Mount(os.Args[1], os.Args[2])
}
