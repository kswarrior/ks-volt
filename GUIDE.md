# KS-Volt Language Guide

Welcome to **KS-Volt**, a high-performance, ahead-of-time (AOT) compiled programming language designed for building reactive web panels and concurrent systems. KS-Volt features a Go-based compiler frontend and a C runtime with an M:N work-stealing scheduler.

## Table of Contents
1. [Getting Started](#getting-started)
2. [Basic Syntax](#basic-syntax)
3. [Ownership & Borrowing](#ownership--borrowing)
4. [Functional Programming](#functional-functions)
5. [The Component System](#hyper-static-components)
6. [Web Engine (KS Panel)](#web-platform-engine)
7. [Concurrency & Tasks](#concurrency--scheduler)
8. [Polyglot Blocks](#polyglot-engine)
9. [File System Macros](#file-system-macros)

---

## Getting Started

### Compilation
To compile a KS-Volt script into a native static binary:
```bash
go run main.go path/to/script.kv
```

### Watch Mode
For reactive development, use watch mode to recompile on file save:
```bash
go run main.go watch path/to/script.kv
```

---

## Basic Syntax

KS-Volt uses a minimalist syntax without variable declaration keywords like `let` or `const`.

### Variables & Data Types
```kv
name = "Jules"        // String
age = 25              // Integer
is_admin = true       // Boolean
list = [1, 2, 3]      // Array
data = json_parse("{\"key\": \"value\"}") // Map
```

### Control Flow
```kv
if (age + 5 == 30) {
    print("Success")
} else {
    print("Fail")
}

loop item in list {
    print(item)
}
```

### Exception Handling
```kv
try {
    file_write("/tmp/test.txt", "content")
} catch(err) {
    print(`Failed: ${err}`)
}
```

---

## Ownership & Borrowing

KS-Volt uses an ownership model inspired by Rust to ensure memory safety without a garbage collector.

### Move Semantics
When you assign one variable to another, the value is **moved**.
```kv
a = "resource"
b = a          // 'a' is now moved
// print(a)   // Error: use of moved value 'a'
```

### Borrowing
Use `&` to borrow a value without taking ownership. Use `&mut` for mutable borrows.
```kv
a = "resource"
b = &a         // 'b' is a borrow
print(a)       // OK: 'a' still owns the resource
```

---

## Hyper-Static Components

Components are optimized for zero-allocation HTML rendering.

### Defining a Component
```kv
component Sidebar(user) {
    `<div class="sidebar">Welcome, ${user}</div>`
}
```

### Importing Components
```kv
import_component "ui.kv" as UI
import_component "widgets/*.kv" as Widgets // Wildcard globbing

UI.Sidebar("Admin")
Widgets_Clock_Clock() // Namespaced call for Widgets/Clock.kv
```

---

## Web Platform Engine

The `web_block` directive defines an isolated multi-threaded HTTP server.

### Basic Routing
```kv
web_block "panel" {
    before_each -> {
        print("Incoming request...")
    }

    path("/") -> {
        `<h1>Welcome to KS Panel</h1>`
    }

    path("/login") -> {
        user = request_form("username")
        redirect("/dashboard")
    }
}
```

### Fragment Rendering (SPA Mode)
Use `render_fragment()` to stream raw HTML fragments for client-side swapping.
```kv
path("/api/stats") -> {
    render_fragment()
    `<div>CPU: 5%</div>` // Streams raw <div> without HTML wrappers or headers
}
```

### Built-in Web Primitives
- `request_header(name)`: Extract HTTP headers.
- `redirect(url)`: Set 302 status and Location header.
- `json(structure)`: Send JSON response with correct mime-type.
- `request_form(key)`: Extract POST form data.
- `request_json()`: Parse request body as JSON.

---

## Concurrency & Scheduler

KS-Volt uses a GMP-style scheduler to map tasks to hardware threads.

### Spawning Tasks
```kv
spawn {
    print("Task running in parallel")
}

interval(1000) {
    print("Heartbeat every second")
}
```

### Background Jobs
Inside a web route, dispatch non-blocking jobs:
```kv
path("/backup") -> {
    dispatch_job() -> {
        print("Starting backup...")
        sleep(5)
        print("Backup complete")
    }
    `Backup started in background.`
}
```

---

## Polyglot Engine

Directly embed code from other languages within your KV scripts.

### Go Block
```kv
go_block {
    func NativeSum(a, b int) int {
        return a + b
    }
}
```

### JavaScript Block (QuickJS)
```kv
js_block {
    console.log("Hello from QuickJS");
}
```

### Python Block
```kv
py_block {
    import math
    print(f"PI is {math.pi}")
}
```

---

## File System Macros

These map directly to POSIX system calls for maximum performance.

- `fs_rm(path)`: Unlink a file.
- `fs_mv(src, dst)`: Rename/move a file.
- `fs_cp(src, dst)`: Streaming copy.
- `fs_touch(path)`: Create empty file or update timestamp.
- `fs_cat(path)`: Stream file content to stdout.
