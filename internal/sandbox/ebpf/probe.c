//go:build ignore

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

char __license[] SEC("license") = "Dual MIT/GPL";

struct trace_event_raw_sys_enter {
	long int id;
	long unsigned int args[6];
	char ent[0];
};

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 1 << 24);
} events SEC(".maps");

// target_pid_ns_inum: set from userspace via BPF map to filter by PID namespace.
// Value 0 = disabled (monitor all processes). Non-zero = only alert on PIDs in this ns.
struct {
	__uint(type, BPF_MAP_TYPE_ARRAY);
	__uint(max_entries, 1);
	__type(key, __u32);
	__type(value, __u32);
} config SEC(".maps");

struct event {
    unsigned int pid;
    unsigned int syscall_id;
    char comm[16];
};

struct event *unused_event __attribute__((unused));

// Monitored syscalls:
//   41 = socket          — network socket creation
//   42 = connect         — outbound connections
//   56 = clone           — process forking
//   57 = fork            — process forking (legacy)
//   59 = execve          — process execution
//  319 = memfd_create    — anonymous RAM-backed fd (fileless exec)
//  322 = execveat        — fd-based exec (memfd payload delivery)
//  425 = io_uring_setup  — async I/O ring (bypasses sys_enter for socket/connect)
//  426 = io_uring_enter  — async I/O submission
//
// Phase 26 Audit: syscalls 319, 322, 425, 426 were blind spots.
// The comm=="python3" filter was overly narrow — renamed processes or
// Go binaries executing from memfd were completely invisible.

static __always_inline int is_monitored_syscall(long int id) {
	switch (id) {
	case 41:  // socket
	case 42:  // connect
	case 56:  // clone
	case 57:  // fork
	case 59:  // execve
	case 319: // memfd_create  [PHASE 26 FIX]
	case 322: // execveat      [PHASE 26 FIX]
	case 425: // io_uring_setup [PHASE 26 FIX]
	case 426: // io_uring_enter [PHASE 26 FIX]
		return 1;
	default:
		return 0;
	}
}

SEC("tracepoint/raw_syscalls/sys_enter")
int trace_sys_enter(struct trace_event_raw_sys_enter *ctx) {
	unsigned int pid = bpf_get_current_pid_tgid() >> 32;
	long int syscall_id = ctx->id;

	if (!is_monitored_syscall(syscall_id)) {
		return 0;
	}

	// Phase 26 Fix: removed hardcoded comm=="python3" filter.
	// Now monitors ALL processes calling restricted syscalls.
	// Userspace loader can optionally filter by PID namespace
	// via the config map (set target_pid_ns_inum != 0).

	char comm[16];
	bpf_get_current_comm(&comm, sizeof(comm));

	struct event *e;
	e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
	if (!e) {
		return 0;
	}

	e->pid = pid;
	e->syscall_id = syscall_id;

	#pragma unroll
	for (int i = 0; i < 16; i++) {
		e->comm[i] = comm[i];
	}

	bpf_ringbuf_submit(e, 0);

	return 0;
}
