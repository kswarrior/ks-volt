# 🧪 KS-Volt Testing Suite

This directory contains comprehensive tests to verify the core functionality and performance of the KS-Volt compiler and runtime.

## 📁 Test Categories

*   **[Panel](panel/)**: A full-stack web panel demonstration with login and dashboard routes.
*   **[Polyglot](polyglot/)**: Verification of inline Go, Rust, JavaScript, and Python blocks.
*   **[FS Macros](fs/)**: Validation of bare-metal POSIX file system shortcuts.
*   **[Concurrency](concurrency/)**: Stress test for the GMP Work-Stealing Scheduler and `spawn/interval` primitives.
*   **[Exceptions](exceptions/)**: Verification of `try/catch` guardrails and `setjmp/longjmp` trapping.

## 🚀 Running the Tests

Ensure you have the `kv` binary built at the root:
```bash
go build -o kv main.go
```

### Run individual tests:

```bash
# Panel Server
./kv testing/panel/main.kv

# Polyglot Block Verification
./kv testing/polyglot/test.kv

# FS Macro Validation
./kv testing/fs/test.kv

# Concurrency & Scheduler Test
./kv testing/concurrency/test.kv

# Exception Guardrail Test
./kv testing/exceptions/test.kv
```
