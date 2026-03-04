#ifndef __BPF_HELPERS_H
#define __BPF_HELPERS_H

#define SEC(name) \
_Pragma("GCC diagnostic push") \
_Pragma("GCC diagnostic ignored \"-Wignored-attributes\"") \
__attribute__((section(name), used)) \
_Pragma("GCC diagnostic pop")

#define __uint(name, val) int (*name)[val]
#define __type(name, val) typeof(val) *name
#define BPF_MAP_TYPE_RINGBUF 27

typedef unsigned int __u32;
typedef unsigned long long __u64;

static void *(*bpf_ringbuf_reserve)(void *ringbuf, __u64 size, __u64 flags) = (void *) 131;
static void (*bpf_ringbuf_submit)(void *data, __u64 flags) = (void *) 132;
static __u64 (*bpf_get_current_pid_tgid)(void) = (void *) 14;
static int (*bpf_get_current_comm)(void *buf, __u32 size_of_buf) = (void *) 16;
#endif
