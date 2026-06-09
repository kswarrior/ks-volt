# ⚡ KS-Volt

KS-Volt is an ultra-high-performance, minimalist programming language designed for bare-metal speed and maximum concurrency. It features a clean syntax inspired by Python but compiles directly to **Static Native Machine Binaries** via a highly optimized C runtime.

## 🚀 Key Features

- **True AOT Compilation**: Compiles `.kv` files into standalone, statically linked binaries with ZERO runtime dependencies.
- **GMP-Style Work-Stealing Scheduler**: Custom Go-style non-blocking scheduler implemented in C for optimal multi-core utilization.
- **Sub-Megabyte Memory Footprint**: Aggressive memory management and OS-level release (malloc_trim) targeting sub-1MB idle RAM usage.
- **Blazing Fast**: Native execution speed with GCC -O3 optimizations.
- **Maximum Concurrency**: Lightweight tasks (`spawn`) mapped to hardware threads via worker rings.
- **Built-in Web Engine**: High-performance static socket-based web server.
- **Embedded Local Database**: Persistent key-value storage.
- **Raw Networking**: Native support for socket connections and bot streaming.

## 🛠️ Language Syntax

### Variable Assignment
Assignments are implicit. No keywords required.
```kv
PORT = 8080
STATUS = "ACTIVE"
```

### Concurrency with `spawn`
Execute any block of code in a lightweight task managed by the work-stealing scheduler.
```kv
spawn start_service() {
    print("Service is running...")
}
```

### String Interpolation
Dynamic string concatenation using the `+` operator (mapped to `dynamic_strcat`).
```kv
print("⚡ Server spinning up on port: " + PORT)
```

### Embedded Database
Simple `db_save` and `db_get` for persistent storage.
```kv
db_save("state", "online")
current = db_get("state")
```

### Raw Networking
Asynchronous TCP connections for bots or custom socket protocols.
```kv
spawn connect_bot("127.0.0.1", 25565) {
    print("🤖 Bot connected!")
}
```

## 📈 Performance & Load

KS-Volt is built for extreme efficiency:
- **Low Latency**: Direct hardware thread mapping.
- **Memory**: Drastic reduction in idle RAM compared to Go/Node.js.
- **Compilation**: Fast AOT compilation via the KS-Volt toolchain.

## 🏁 Getting Started

### Prerequisites
- Go (for the compiler frontend)
- GCC (for the AOT backend)

### Compilation
To compile a KS-Volt script (`.kv` file) to a static native binary:

```bash
go run main.go path/to/your_script.kv
```

### Execution
Run the generated standalone binary directly:

```bash
./your_script
```

## 📂 Project Structure

- `token/`: Token definitions.
- `lexer/`: Stateful scanner.
- `ast/`: Abstract Syntax Tree nodes.
- `parser/`: Recursive descent parser.
- `compiler/`: C-Targeted AOT Transpiler and Work-Stealing Scheduler.
- `main.go`: CLI entry point and GCC build orchestrator.
