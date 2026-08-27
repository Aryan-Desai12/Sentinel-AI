package collector

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type NetworkStats struct {
	Received    uint64
	Transmitted uint64
}

type NetworkCollector struct {
	previous    NetworkStats
	initialized bool
}

//Here we are calculating(Network speed/rate) how much the rate at which data is being transferred through a network interface. It is different from Internet speed which usually means the rate at which your machine is transferring data to/from the Internet.
//you're measuring actual network traffic rate, not the Internet connection's maximum speed.

func readNetworkStats() (NetworkStats, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return NetworkStats{}, fmt.Errorf("open /proc/net/dev: %w", err)
	}
	defer file.Close()

	var stats NetworkStats

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip headers.
		if !strings.Contains(line, ":") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		fields := strings.Fields(parts[1])

		if len(fields) < 9 {
			continue
		}

		received, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			return NetworkStats{}, fmt.Errorf("parse received bytes: %w", err)
		}

		transmitted, err := strconv.ParseUint(fields[8], 10, 64)
		if err != nil {
			return NetworkStats{}, fmt.Errorf("parse transmitted bytes: %w", err)
		}

		stats.Received += received
		stats.Transmitted += transmitted
	}

	if err := scanner.Err(); err != nil {
		return NetworkStats{}, fmt.Errorf("read /proc/net/dev: %w", err)
	}

	return stats, nil
}

func NewNetworkCollector() *NetworkCollector {
	return &NetworkCollector{}
}

func (c *NetworkCollector) Collect() (NetworkStats, error) {
	current, err := readNetworkStats()
	if err != nil {
		return NetworkStats{}, err
	}

	// First reading establishes the baseline.
	if !c.initialized {
		c.previous = current
		c.initialized = true

		return NetworkStats{}, nil
	}

	// Detect counter reset.
	if current.Received < c.previous.Received ||
		current.Transmitted < c.previous.Transmitted {
		c.previous = current
		return NetworkStats{}, fmt.Errorf("network counters reset")
	}

	stats := NetworkStats{
		Received:    current.Received - c.previous.Received,
		Transmitted: current.Transmitted - c.previous.Transmitted,
	}

	c.previous = current

	return stats, nil
}
