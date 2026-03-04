# Sandbox & Isolation Requirements

Математический агент (Math_Verifier_Agent) выполняет недоверенный (или сгенерированный) код для проверки доказательств. Это наивысшая точка риска (RCE by design). 

## Isolation Layers (Defense in Depth)

1. **Kernel Layer (Namespaces)**: Изолированный PID namespace. Нет доступа к сети (NET namespace empty loopback). 
2. **Filesystem**: Read-Only mount (chroot/OverlayFS). Запрет доступа к /sys, /proc (hide pid).
3. **Resource Limits (cgroups v2)**:
   - Макс RAM: 256 MB.
   - Макс CPU: 1 Core.
   - PIDs max: 10 (защита от fork-bomb).
4. **Syscall Filtering (Seccomp-BPF)**:
   - Allow: read, write, exit, brk, mmap, rtsig.
   - Deny: execve (после старта python/sympy), network syscalls, clone.

## Attack Surface Mitigation
Любой сгенерированный/полученный Python код сначала парсится через `ast.parse()` в Go перед отправкой в Sandbox для гарантии отсутствия попыток чтения локальных файлов (`open`, `__import__`).
