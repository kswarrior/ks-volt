# ⚡ KS-Volt: Definitive Edition

KS-Volt is an ultra-high-performance, minimalist programming language designed for bare-metal speed and maximum concurrency. It features a clean syntax inspired by Python but compiles directly to **Static Native Machine Binaries** via a feature-complete C runtime.

## 🚀 Definitive Features

- **True AOT Compilation**: Compiles `.kv` files into standalone, statically linked binaries with ZERO runtime dependencies.
- **M:N Work-Stealing Scheduler**: Custom green-thread scheduler implemented in C for optimal multi-core load balancing.
- **Unmanaged Collections**: Zero-allocation contiguous arrays and vector loops.
- **Sub-Megabyte Memory Footprint**: Aggressive memory management targeting sub-1MB idle RAM usage.
- **Zero-Overhead Exception Guardrails**: High-speed error trapping using `try/catch` with non-local jumps (`setjmp/longjmp`).
- **Micro-JSON Scanner**: Reflection-free, high-speed JSON manipulation.
- **Built-in Web & Networking**: Native support for socket connections, bot streaming, and static web serving.

## 🛠️ Language Syntax

### Variable Assignment & Conditionals
Implicit assignments and native control flow.
```kv
IS_SECURE = true
if (IS_SECURE) {
    print("Security active")
}
```

### Unmanaged Arrays & Loops
Efficient contiguous memory access.
```kv
ports = [8080, 8081, 8082]
loop p in ports {
    print("Port: " + p)
}
```

### Exception Handling
Catch OS execution exceptions without process termination.
```kv
try {
    file_write("log.txt", data)
} catch (err) {
    print("Error: " + err)
}
```

### JSON Manipulation
High-speed micro-JSON parsing.
```kv
data = json_parse("{\"id\": 101}")
print(data["id"])
```

## 📈 Performance Specifications

- **Latency**: Sub-microsecond task switching via green threads.
- **Binary Size**: Minimal footprint through symbol stripping (`-s`) and debug disable (`-w`).
- **Static Linking**: All dependencies baked into the single machine asset.

## 🏁 Getting Started

### Prerequisites
- Go (Compiler Frontend)
- GCC (AOT Backend)

### Compile & Run
```bash
go run main.go script.kv
./script
```

## 📂 Project Structure

- `token/`: Token definitions.
- `lexer/`: Stateful scanner.
- `ast/`: Abstract Syntax Tree.
- `parser/`: Recursive descent parser.
- `compiler/`: Definitive C-Targeted AOT Compiler & GMP Scheduler.
- `main.go`: GCC build orchestrator.
