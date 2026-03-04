#include <stdio.h>
#include <stdlib.h>
#include <sys/mman.h>
#include <string.h>
#include <unistd.h>

int main() {
    // 1. Allocate Anonymous RWX memory (Highly suspicious pattern)
    void *exec_mem = mmap(NULL, 4096, PROT_READ | PROT_WRITE | PROT_EXEC, 
                          MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
    if (exec_mem == MAP_FAILED) {
        perror("mmap");
        return 1;
    }

    // 2. Inject a fake NOP sled shellcode pattern
    char shellcode[] = "\x90\x90\x90\x90\xcc\xcc\xcc\xcc"; 
    memcpy(exec_mem, shellcode, sizeof(shellcode));

    printf("Injected RWX anomaly at %p. PID: %d. Sleeping to allow EDR to scan...\n", exec_mem, getpid());

    // 3. Keep process alive so EDR can find it
    sleep(10); 

    return 0;
}
