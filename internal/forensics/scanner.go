package forensics

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// MemoryRegion represents a single line parsed from /proc/PID/maps
type MemoryRegion struct {
	StartAddr uint64
	EndAddr   uint64
	Perms     string // e.g., "rwxp", "r-xp"
	Offset    uint64
	Dev       string
	Inode     uint64
	Path      string // The backing file, or empty/pseudo (e.g. [heap], [stack])
}

// Scanner orchestrates reading RAM layout for running processes.
type Scanner struct{}

func NewScanner() *Scanner {
	return &Scanner{}
}

// FindAnomalousRegions scans a target Process ID for unusual memory allocations,
// primarily focusing on Anonymous RWX (Read-Write-Execute) segments which are
// classic indicators of Reflective DLL Injection, JIT Sprays, or raw Shellcode.
func (s *Scanner) FindAnomalousRegions(pid int) ([]MemoryRegion, error) {
	mapsPath := fmt.Sprintf("/proc/%d/maps", pid)
	file, err := os.Open(mapsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open memory maps for PID %d: %w", pid, err)
	}
	defer file.Close()

	var anomalies []MemoryRegion
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		region, err := parseMapLine(line)
		if err != nil {
			log.Printf("[EDR-WARN] Failed parsing maps line '%s': %v", line, err)
			continue
		}

		// Heuristic 1: RWX Permissions. (Memory should almost never be both Writable and Executable)
		// Heuristic 2: Anonymous backing (Path == ""). Shellcode doesn't usually have a backing file.
		isRWX := strings.HasPrefix(region.Perms, "rwx")
		isAnonymous := region.Path == "" || strings.HasPrefix(region.Path, "[anon")

		if isRWX && isAnonymous {
			// This is highly suspicious.
			anomalies = append(anomalies, region)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading maps for PID %d: %w", pid, err)
	}

	return anomalies, nil
}

// parseMapLine parses a line from /proc/[pid]/maps
// Format: 00400000-00452000 r-xp 00000000 08:02 173521      /usr/bin/dbus-daemon
func parseMapLine(line string) (MemoryRegion, error) {
	var region MemoryRegion
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return region, fmt.Errorf("malformed line: %s", line)
	}

	// 1. Parse Address Range: 00400000-00452000
	addrs := strings.Split(fields[0], "-")
	if len(addrs) != 2 {
		return region, fmt.Errorf("malformed address string")
	}

	start, err := strconv.ParseUint(addrs[0], 16, 64)
	if err != nil {
		return region, err
	}
	end, err := strconv.ParseUint(addrs[1], 16, 64)
	if err != nil {
		return region, err
	}

	region.StartAddr = start
	region.EndAddr = end

	// 2. Parse Perms (e.g. r-xp)
	region.Perms = fields[1]

	// 3. Parse Offset
	offset, err := strconv.ParseUint(fields[2], 16, 64)
	if err != nil {
		return region, err
	}
	region.Offset = offset

	// 4. Parse Device
	region.Dev = fields[3]

	// 5. Parse Inode
	inode, err := strconv.ParseUint(fields[4], 10, 64)
	if err != nil {
		return region, err
	}
	region.Inode = inode

	// 6. Parse Path (Optional, some regions are anonymous)
	if len(fields) >= 6 {
		region.Path = strings.Join(fields[5:], " ")
	} else {
		region.Path = ""
	}

	return region, nil
}
