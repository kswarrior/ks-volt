# ⚡ KS-Volt

KS-Volt is a high-performance, minimalist programming language designed for maximum concurrency, blazing fast development, and native Go performance. It features a clean syntax inspired by Python but leverages Go's powerful runtime for goroutines, networking, and more.

## 🚀 Key Features

- **True AOT Compilation**: Compiles `.kv` files directly into standalone machine binaries.
- **Simpler than Python**: No variable keywords like `let` or `var`. Minimalist block syntax.
- **Blazing Fast**: Instant compilation and native execution performance.
- **Maximum Concurrency**: Exposes Go's native goroutines via the `spawn` keyword.
- **Built-in Web Engine**: Effortlessly serve HTML and interact with APIs.
- **Embedded Local Database**: Persistent, thread-safe key-value storage out of the box.
- **Raw Networking**: Native support for socket connections and bot streaming.

## 🛠️ Language Syntax

### Variable Assignment
Assignments are implicit. No keywords required.
```kv
PORT = 8080
STATUS = "ACTIVE"
```

### Concurrency with `spawn`
Execute any block of code in a native Go goroutine.
```kv
spawn start_service() {
    print("Service is running...")
}
```

### String Interpolation
Dynamic string concatenation using the `+` operator.
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

KS-Volt is built on top of the Go standard library, inheriting its world-class performance:
- **Concurrency**: Can easily handle tens of thousands of concurrent `spawn` blocks (goroutines).
- **Compilation**: Near-instant execution.
- **Networking**: High-throughput `net/http` and `net` stack exposure.
- **Memory**: Efficient memory management with Go's garbage collector.

## 🏁 Getting Started

To compile a KS-Volt script (`.kv` file):

```bash
go run main.go path/to/your_script.kv
```

This will generate a standalone binary (e.g., `test`) that you can execute directly:

```bash
./test
```

## 📂 Project Structure

- `token/`: Token definitions.
- `lexer/`: Stateful scanner.
- `ast/`: Abstract Syntax Tree nodes.
- `parser/`: Recursive descent parser.
- `compiler/`: AOT Transpiler logic.
- `main.go`: CLI entry point and build orchestrator.
