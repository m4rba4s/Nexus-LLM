package ebpf

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -Wall -Werror -I." bpf probe.c

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
)

// Event struct matching the C struct layout
type Event struct {
	PID       uint32
	SyscallID uint32
	Comm      [16]byte
}

type SandboxMonitor struct {
	objs   bpfObjects
	links  []link.Link
	reader *ringbuf.Reader
	Alerts chan string
}

func NewSandboxMonitor() (*SandboxMonitor, error) {
	var objs bpfObjects
	err := loadBpfObjects(&objs, nil)
	if err != nil {
		return nil, err
	}

	// Attach tracepoint
	tp, err := link.Tracepoint("raw_syscalls", "sys_enter", objs.TraceSysEnter, nil)
	if err != nil {
		objs.Close()
		return nil, err
	}

	// Open ring buffer
	rd, err := ringbuf.NewReader(objs.Events)
	if err != nil {
		tp.Close()
		objs.Close()
		return nil, err
	}

	monitor := &SandboxMonitor{
		objs:   objs,
		links:  []link.Link{tp},
		reader: rd,
		Alerts: make(chan string, 10),
	}

	go monitor.listen()

	return monitor, nil
}

func (m *SandboxMonitor) listen() {
	log.Println("[eBPF] Starting ring-0 syscall surveillance on python3.")
	var event Event
	for {
		record, err := m.reader.Read()
		if err != nil {
			if err == ringbuf.ErrClosed {
				return
			}
			continue
		}

		if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &event); err != nil {
			continue
		}

		comm := string(bytes.TrimRight(event.Comm[:], "\x00"))
		syscallStr := ""
		switch event.SyscallID {
		case 41, 42:
			syscallStr = "network (socket/connect)"
		case 56, 57:
			syscallStr = "process fork/clone"
		case 59:
			syscallStr = "exec"
		case 319:
			syscallStr = "fileless exec (memfd_create)"
		case 322:
			syscallStr = "fd-based exec (execveat)"
		case 425:
			syscallStr = "io_uring setup (async I/O ring)"
		case 426:
			syscallStr = "io_uring enter (async submission)"
		default:
			syscallStr = "restricted system call"
		}

		alert := "[eBPF ALERT] Process: " + comm + " attempting illegal " + syscallStr + "!"
		m.Alerts <- alert
	}
}

func (m *SandboxMonitor) Close() {
	m.reader.Close()
	for _, l := range m.links {
		l.Close()
	}
	m.objs.Close()
	close(m.Alerts)
}

// LoadAndAttach compiles raw C code into eBPF bytecode using clang, and attaches it dynamically.
func (m *SandboxMonitor) LoadAndAttach(ctx context.Context, bpfCode string) error {
	log.Println("[eBPF-HOT-RELOAD] Compiling new defense probe on-the-fly...")

	tmpPath := "/tmp/nexus_dynamic_probe.c"
	objPath := "/tmp/nexus_dynamic_probe.o"

	if err := os.WriteFile(tmpPath, []byte(bpfCode), 0644); err != nil {
		return fmt.Errorf("failed to write dynamic probe: %w", err)
	}

	cmd := exec.CommandContext(ctx, "clang", "-O2", "-g", "-Wall", "-Werror", "-target", "bpf", "-c", tmpPath, "-o", objPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("clang compilation failed: %v\nOutput: %s", err, string(output))
	}

	spec, err := ebpf.LoadCollectionSpec(objPath)
	if err != nil {
		return fmt.Errorf("failed to load ELF spec: %w", err)
	}

	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return fmt.Errorf("failed to load BPF collection: %w", err)
	}

	var attachedCount int
	for name, progSpec := range spec.Programs {
		prog := coll.Programs[name]
		if prog == nil {
			continue
		}

		if strings.HasPrefix(progSpec.SectionName, "tracepoint/") {
			parts := strings.SplitN(strings.TrimPrefix(progSpec.SectionName, "tracepoint/"), "/", 2)
			if len(parts) == 2 {
				tp, err := link.Tracepoint(parts[0], parts[1], prog, nil)
				if err != nil {
					log.Printf("[eBPF-HOT-RELOAD] Failed to attach tracepoint %s: %v", progSpec.SectionName, err)
					continue
				}
				m.links = append(m.links, tp)
				attachedCount++
				log.Printf("[eBPF-HOT-RELOAD] Attached dynamic tracepoint: %s (Prog: %s)", progSpec.SectionName, name)
			}
		} else if strings.HasPrefix(progSpec.SectionName, "kprobe/") {
			funcName := strings.TrimPrefix(progSpec.SectionName, "kprobe/")
			kp, err := link.Kprobe(funcName, prog, nil)
			if err != nil {
				log.Printf("[eBPF-HOT-RELOAD] Failed to attach kprobe %s: %v", funcName, err)
				continue
			}
			m.links = append(m.links, kp)
			attachedCount++
			log.Printf("[eBPF-HOT-RELOAD] Attached dynamic kprobe: %s (Prog: %s)", funcName, name)
		}
	}

	if attachedCount == 0 {
		return fmt.Errorf("no attachable BPF programs (tracepoints/kprobes) found in dynamic object")
	}

	log.Printf("[eBPF-HOT-RELOAD] Successfully hot-reloaded %d probes into Ring-0!", attachedCount)
	return nil
}
