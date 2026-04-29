// SPDX-License-Identifier: EUPL-1.2
//
// Wanderer egress-flow eBPF program. Attaches a tracepoint on
// `sys_enter_connect` and emits a perf event per connect() call:
// {pid, comm, family, daddr/daddr6, dport}.
//
// Build path (NOT yet wired into go generate):
//
//   clang -O2 -target bpf -D__TARGET_ARCH_x86 \
//     -I/usr/include/x86_64-linux-gnu \
//     -c connect.bpf.c -o connect.bpf.o
//   bpf2go -tags linux Connect connect.bpf.c
//
// The bpf2go integration is deferred (see ADR-0010 + the archived
// add-egress-flow-probe tasks notes); this file ships as the
// auditable C source for future review.

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

char __license[] SEC("license") = "EUPL-1.2";

#define AF_INET  2
#define AF_INET6 10

struct flow_event {
    __u32 pid;
    __u8  family;
    __u8  pad[3];
    __u16 dport;
    __u8  daddr_v6[16];   // for AF_INET, only first 4 bytes are used
    __u8  comm[16];
};

struct {
    __uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
    __uint(key_size, sizeof(int));
    __uint(value_size, sizeof(int));
} events SEC(".maps");

// sys_enter_connect tracepoint ABI: int sys_enter_connect(int fd,
// const struct sockaddr __user *uservaddr, int addrlen).
// The tracepoint argument struct is laid out as the syscall's
// arguments after the common header.
struct sys_enter_connect_args {
    unsigned long long _pad;
    int                __syscall_nr;
    __u64              fd;
    __u64              uservaddr;
    __u64              addrlen;
};

SEC("tracepoint/syscalls/sys_enter_connect")
int trace_connect(struct sys_enter_connect_args *ctx) {
    struct sockaddr addr;
    if (!ctx->uservaddr) {
        return 0;
    }
    if (bpf_probe_read_user(&addr, sizeof(addr), (void *)ctx->uservaddr) != 0) {
        return 0;
    }
    // We only care about IPv4 / IPv6. Skip AF_UNIX and friends so we
    // do not flood the perf ring with local socket noise.
    if (addr.sa_family != AF_INET && addr.sa_family != AF_INET6) {
        return 0;
    }

    struct flow_event ev = {0};
    ev.pid = bpf_get_current_pid_tgid() >> 32;
    ev.family = addr.sa_family;
    bpf_get_current_comm(&ev.comm, sizeof(ev.comm));

    if (addr.sa_family == AF_INET) {
        struct sockaddr_in v4;
        bpf_probe_read_user(&v4, sizeof(v4), (void *)ctx->uservaddr);
        ev.dport = bpf_ntohs(v4.sin_port);
        __builtin_memcpy(ev.daddr_v6, &v4.sin_addr, 4);
    } else {
        struct sockaddr_in6 v6;
        bpf_probe_read_user(&v6, sizeof(v6), (void *)ctx->uservaddr);
        ev.dport = bpf_ntohs(v6.sin6_port);
        __builtin_memcpy(ev.daddr_v6, &v6.sin6_addr, 16);
    }

    bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, &ev, sizeof(ev));
    return 0;
}
