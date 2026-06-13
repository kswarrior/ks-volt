# ⚡ KS-Volt: The Definitive Unified Polyglot Engine

KS-Volt is an ultra-high-performance, compiled programming language designed for zero-allocation efficiency, native concurrency, and seamless multi-language integration. It transpiles directly to optimized C, utilizing a custom GMP-style Work-Stealing Scheduler and a sub-megabyte RAM footprint.

## 🚀 Key Features

*   **Unified Polyglot Architecture**: Execute Go, Rust, JavaScript, and Python inline within a single `.kv` project.
*   **Hyper-Static Components**: Build modular, reusable UI components with zero runtime rendering overhead.
*   **Native Web Routing**: Integrated DSL for high-velocity web servers and WebSocket controllers.
*   **Bare-Metal Performance**: Direct mapping to POSIX system calls and unmanaged memory management.
*   **GMP-Style Concurrency**: A lightweight green-thread scheduler that scales across all CPU cores.
*   **AOT Compilation**: Produces a single, dependency-free static binary.

## 🛠️ Quick Start

### Prerequisites
*   Go (for the compiler)
*   GCC (for native binary generation)
*   Optional: TinyGo/Go (for Go blocks), Rust/Cargo (for Rust blocks), Python3-dev (for Python blocks).

### Installation
```bash
git clone https://github.com/ks-volt/ks-volt.git
cd ks-volt
go build -o kv main.go
```

### Run a Script
```bash
./kv my_script.kv
./my_script
```

### Watch Mode (Developer Velocity)
```bash
./kv watch my_script.kv
```

## 📖 Documentation

*   [**Backend & Runtime**](BACKEND.md): Native compilation, scheduler, and system primitives.
*   [**Frontend & Components**](FRONTEND.md): Hyper-static components and web routing engine.
*   [**Polyglot Blocks**](POLYGLOT.md): Integrating Go, Rust, JS, and Python.
*   [**Performance Guide**](PERFORMANCE.md): Resource utilization and optimization metrics.

## 📜 License
MIT License
