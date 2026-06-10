# ⚡ KS-Volt: Definitive Edition

KS-Volt is an ultra-high-performance, minimalist programming language designed for bare-metal speed and maximum concurrency. It features a clean syntax inspired by Python but compiles directly to **Static Native Machine Binaries** via a feature-complete C runtime.

## 🚀 Definitive Features

- **True AOT Compilation**: Compiles `.kv` files into standalone, statically linked binaries with ZERO runtime dependencies.
- **M:N Work-Stealing Scheduler**: Custom green-thread scheduler implemented in C for optimal multi-core load balancing.
- **First-Class Functions**: Define, pass, and execute functions dynamically.
- **Unmanaged Collections**: Zero-allocation contiguous arrays and vector loops.
- **Native String Methods**: Built-in `.trim()` and `.upper()` for raw character manipulation.
- **Pointer Tracking**: Explicit hardware memory address referencing via `get_addr`.
- **Sub-Megabyte Memory Footprint**: Aggressive memory management targeting sub-1MB idle RAM usage.
- **Zero-Overhead Exception Guardrails**: High-speed error trapping using `try/catch` with non-local jumps (`setjmp/longjmp`).
- **Micro-JSON Scanner**: Reflection-free, high-speed JSON manipulation.
- **Built-in Web & Networking**: Native support for socket connections, bot streaming, and static web serving.

## 🛠️ Language Syntax

### Variable Assignment & Conditionals
```kv
IS_SECURE = true
if (IS_SECURE) {
    print("Security active")
}
```

### First-Class Functions
```kv
fn greet(name) {
    return "Hello, " + name
}
print(greet("KS-Volt"))
```

### Unmanaged Arrays & Loops
```kv
ports = [25565, 25566, 25567]
loop p in ports {
    print("Port: " + p)
}
```

### Exception Handling
```kv
try {
    file_write("log.txt", data)
} catch (err) {
    print("Error: " + err)
}
```

## 📊 Performance Testing & Monitoring

KS-Volt binaries are designed to be extremely lightweight and fast.

### 1. Check Binary Footprint
To verify that the binary is statically linked and stripped of symbols (resulting in a small file size):
```bash
ls -lh test
file test
```

### 2. Monitor RAM Profile
To monitor the sub-megabyte RAM profile while running the definitive `test.kv` orchestrator:
```bash
# In one terminal, run the orchestrator
./test

# In another terminal, check the resident set size (RSS)
ps -o rss,command -p $(pgrep test)
```
Target: **< 1000 KB (Sub-1MB)**

### 3. Execution Speed
The M:N scheduler handles thousands of green threads with sub-microsecond latency. You can observe the background telemetry and bot workers running concurrently without blocking the main event loop.

## 🏁 Getting Started

### Prerequisites
- Go (Compiler Frontend)
- GCC (AOT Backend)

### Compile & Run
```bash
go run main.go test.kv
./test
```

## 📂 Project Structure

- `token/`: Token definitions.
- `lexer/`: Stateful scanner.
- `ast/`: Abstract Syntax Tree.
- `parser/`: Recursive descent parser.
- `compiler/`: Definitive C-Targeted AOT Compiler & GMP Scheduler.
- `main.go`: GCC build orchestrator.
