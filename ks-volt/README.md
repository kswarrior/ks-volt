# ⚡ KS-Volt: Definitive Architecture

KS-Volt is an ultra-high-performance, minimalist programming language designed for bare-metal speed and maximum concurrency. It features a clean syntax inspired by Python but compiles directly to **Static Native Machine Binaries** via a feature-complete C runtime.

## 🚀 Definitive Features

- **True AOT Compilation**: Compiles `.kv` files into standalone, statically linked binaries with ZERO runtime dependencies.
- **Unified Polyglot Engine**: Execute inline **Go, Rust, JavaScript (QuickJS), and Python** code blocks within a single project.
- **Hyper-Static Components**: Ingest sub-components at compile-time with alias resolution for zero runtime rendering overhead.
- **Named Routing Controllers**: Build modular web apps with isolated routing maps and global server-side middleware.
- **Zero-Allocation Stream Buffers**: High-velocity string interpolation optimized for minimal memory footprint.
- **M:N Work-Stealing Scheduler**: Custom green-thread scheduler implemented in C for optimal multi-core load balancing.
- **Native FS Macros**: Abbreviated file system primitives (`fs_rm`, `fs_cp`, etc.) mapping directly to POSIX calls.
- **Sub-Megabyte Memory Footprint**: Aggressive memory management targeting sub-1MB idle RAM usage.
- **Zero-Overhead Exception Guardrails**: High-speed error trapping using `try/catch` with non-local jumps (`setjmp/longjmp`).

## 🛠️ Language Syntax

### Unified Polyglot Blocks
```kv
go_block {
  func InternalProcessor() { ... }
}

rust_block {
  fn calculate_hash() { ... }
}

js_block {
  console.log("QuickJS execution");
}
```

### Static Components & Ingestion
```kv
// In components.kv
component Header(title) {
  print(`--- ${title} ---`)
}

// In main.kv
import_component "components.kv" as UI
render_UIHeader("Main Dashboard")
```

### Named Web Controllers
```kv
web_block "admin_app" {
  before_each() -> {
    print("Security check")
  }
  
  path("/") -> {
    print("Welcome to Admin")
  }
}
```

## 🏁 Getting Started

### Prerequisites
- Go (Compiler Frontend)
- GCC (AOT Backend)
- Python 3.x (Development Headers)
- Cargo (Rust blocks support)

### CLI Commands
```bash
# Compile and run
go run main.go app.kv
./app

# Watch mode for rapid development
go run main.go watch app.kv
```

## 📂 Project Structure

- `token/`: Token definitions.
- `lexer/`: Stateful scanner with raw block support.
- `ast/`: Abstract Syntax Tree with component and routing nodes.
- `parser/`: Recursive descent parser with build-time ingestion.
- `compiler/`: Definitive C-Targeted AOT Compiler & GMP Scheduler.
- `deps/`: Embedded QuickJS engine source.
- `main.go`: GCC build & linker orchestrator with watch mode.
