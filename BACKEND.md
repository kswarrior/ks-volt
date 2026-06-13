# 🏗️ KS-Volt Backend & Runtime Documentation

The KS-Volt backend is a high-performance transpilation engine that converts `.kv` scripts into highly optimized, standalone C source code.

## ⚙️ Native Compilation Pipeline

1.  **Lexical Analysis**: Custom lexer identifies KS-Volt primitives and slices polyglot blocks.
2.  **Recursive Descent Parsing**: Generates an Abstract Syntax Tree (AST) supporting complex nested structures and components.
3.  **C Transpilation**: The compiler maps AST nodes to optimized C patterns, including:
    *   **VoltValue**: A unified tagged union for all dynamic types.
    *   **VoltBuffer**: A zero-allocation string interpolation engine.
    *   **Bare-Metal POSIX Mapping**: Direct calls to `unlink`, `rename`, `fopen`, etc.
4.  **GCC Orchestration**: The compiler invokes `gcc` with `-O3` and `-pthread` to produce the final static binary.

## 🧵 Concurrency & Scheduling

KS-Volt features a **GMP-style Work-Stealing Scheduler** implemented in the C runtime:
*   **Green Threads**: Lightweight tasks scheduled across a pool of hardware-mapped worker threads.
*   **Work Stealing**: Idle workers automatically steal tasks from other processors' deques to maximize CPU utilization.
*   **Spawn Primitive**: `spawn { ... }` launches a new task into the global scheduler.
*   **Intervals**: `spawn interval(ms) { ... }` provides high-precision recurring task execution.

## 📂 File System Macros

Bare-metal POSIX shortcuts for high-velocity I/O:
*   `fs_rm(path)`: Unmanaged `unlink()`.
*   `fs_mv(src, dest)`: Unmanaged `rename()`.
*   `fs_cp(src, dest)`: Streaming buffer copy loop.
*   `fs_touch(path)`: Bare `fopen(a)`.
*   `fs_cat(path)`: Raw text stream to `stdout`.

## 🛡️ Exception Guardrails

Using C's `setjmp` and `longjmp`, KS-Volt provides sub-microsecond error trapping within green threads via the `try { ... } catch(err) { ... }` syntax.

## 📡 System Primitives

*   `db_save(key, val)` / `db_get(key)`: Thread-safe persistence to `volt_db.json`.
*   `file_write(path, data)`: Synchronous native I/O.
*   `json_parse(raw)`: Zero-reflection micro-scanner for JSON objects.
*   `connect_bot(ip, port) { ... }`: Native networking hook for bot integration.
