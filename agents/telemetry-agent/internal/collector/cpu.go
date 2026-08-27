package collector

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type CPUStats struct {
	User    uint64
	Nice    uint64
	System  uint64
	Idle    uint64
	IOWait  uint64
	IRQ     uint64
	SoftIRQ uint64
	Steal   uint64
}

//Collector reads data → calculation interprets it → agent schedules it.

func readCPUStats() (CPUStats, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return CPUStats{}, fmt.Errorf("open /proc/stat: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return CPUStats{}, fmt.Errorf("read /proc/stat: %w", err)
		}
		return CPUStats{}, fmt.Errorf("empty /proc/stat")
	}

	fields := strings.Fields(scanner.Text())

	if len(fields) < 9 || fields[0] != "cpu" {
		return CPUStats{}, fmt.Errorf("invalid CPU data in /proc/stat")
	}

	values := make([]uint64, 8)

	for i := range values {
		value, err := strconv.ParseUint(fields[i+1], 10, 64)
		if err != nil {
			return CPUStats{}, fmt.Errorf("parse CPU value: %w", err)
		}

		values[i] = value
	}

	return CPUStats{
		User:    values[0],
		Nice:    values[1],
		System:  values[2],
		Idle:    values[3],
		IOWait:  values[4],
		IRQ:     values[5],
		SoftIRQ: values[6],
		Steal:   values[7],
	}, nil
}

func calculateCPUUsage(previous, current CPUStats) float64 {
	prevIdle := previous.Idle + previous.IOWait
	currIdle := current.Idle + current.IOWait

	prevTotal := previous.User +
		previous.Nice +
		previous.System +
		previous.Idle +
		previous.IOWait +
		previous.IRQ +
		previous.SoftIRQ +
		previous.Steal

	currTotal := current.User +
		current.Nice +
		current.System +
		current.Idle +
		current.IOWait +
		current.IRQ +
		current.SoftIRQ +
		current.Steal

	totalDelta := currTotal - prevTotal
	idleDelta := currIdle - prevIdle

	if totalDelta == 0 {
		return 0
	}

	return (float64(totalDelta-idleDelta) / float64(totalDelta)) * 100
}

//This function calculates utilization from two CPUStats samples

type CPUCollector struct {
	previous    CPUStats
	initialized bool
}

func NewCPUCollector() *CPUCollector {
	return &CPUCollector{}
}

func (c *CPUCollector) Collect() (float64, error) {
	current, err := readCPUStats()
	if err != nil {
		return 0, err
	}

	// First reading establishes the baseline.
	if !c.initialized {
		c.previous = current
		c.initialized = true

		return 0, nil
	}

	usage := calculateCPUUsage(c.previous, current)

	c.previous = current

	return usage, nil
}

// readCPUStats()
//       ↓
// read Linux counters

// calculateCPUUsage()
//       ↓
// calculate percentage

// agent.go
//       ↓
// decide when to call them
