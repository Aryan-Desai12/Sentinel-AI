package collector

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Memory values are non-negative and can be fairly large:
// MemTotal = several GB
// So: uint64 is appropriate for the raw kernel counters/values.
// We'll convert to float64 only when calculating the percentage.


//Here the readMemoryStats function reads the total and available memory from /proc/meminfo and returns them as uint64 values.
//By system call we are getting the memory statistics from the kernel.

func readMemoryStats() (uint64, uint64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, fmt.Errorf("open /proc/meminfo: %w", err)
	}
	defer file.Close()

	var total uint64
	var available uint64

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())

		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "MemTotal:":
			value, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("parse MemTotal: %w", err)
			}
			total = value

		case "MemAvailable:":
			value, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("parse MemAvailable: %w", err)
			}
			available = value
		}

		if total > 0 && available > 0 {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, fmt.Errorf("read /proc/meminfo: %w", err)
	}

	if total == 0 {
		return 0, 0, fmt.Errorf("MemTotal not found")
	}

	if available > total {
		return 0, 0, fmt.Errorf("MemAvailable is greater than MemTotal")
	}

	return total, available, nil
}



// Here we calculate the memory usage percentage based on the total and available memory.

func CollectMemory() (float64, error) {
	total, available, err := readMemoryStats()
	if err != nil {
		return 0, err
	}

	used := total - available

	usage := (float64(used) / float64(total)) * 100

	return usage, nil
}