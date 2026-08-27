package collector

import (
	"fmt"
	"syscall"

)

// What we are doing here?? We want to know:
// How much disk space is currently being used?
//Unlike CPU and memory, disk information isn't best obtained from /proc.
//Here we used use Linux's filesystem statistics through Go's syscall.Statfs.

// CollectDisk returns the current disk utilization.
// The agent scheduler determines when this function is called.

func CollectDisk() (float64, error) {
	var stat syscall.Statfs_t

	if err := syscall.Statfs("/", &stat); err != nil {  // syscall.Statfs("/", &stat); - Here we ask linux to give me the filesystem statistics for the filesystem on which the root directory / resides.
		return 0, fmt.Errorf("get filesystem stats: %w", err)
	}

	total := stat.Blocks * uint64(stat.Bsize)
	available := stat.Bavail * uint64(stat.Bsize)

	if total == 0 {
		return 0, fmt.Errorf("filesystem has zero total size")
	}

	if available > total {
		return 0, fmt.Errorf("available disk space exceeds total space")
	}

	used := total - available

	usage := (float64(used) / float64(total)) * 100

	return usage, nil
}

// Statfs("/") asks Linux for statistics about the filesystem containing /.
//This Because / is the root of the Linux filesystem hierarchy.
// Think of Linux's filesystem like this:
// /
// ├── home/
// ├── etc/
// ├── var/
// ├── usr/
// ├── tmp/

// We then convert blocks into bytes:
// blocks × block size = bytes
// and calculate:
// used = total - available
// disk usage = used / total × 100