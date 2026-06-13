# 🌍 KS-Volt Polyglot Engine

KS-Volt's Unified Polyglot Engine allows you to embed Go, Rust, JavaScript, and Python code directly inside your `.kv` files. Each block is compiled or linked into the final native binary.

## 🐹 Go Blocks (`go_block`)

Go blocks are compiled into static C archives using standard Go tools.

### Example
```kv
go_block {
    func NativeHash(s string) string {
        return "hashed_" + s
    }
}
```

### Mechanism
1.  Extracts code to `volt_bridge.go`.
2.  Appends `import "C"` and `//export` tags.
3.  Executes `go build -buildmode=c-archive`.
4.  Statically links the resulting `.a` file via GCC.

---

## 🦀 Rust Blocks (`rust_block`)

Rust blocks leverage Cargo for high-performance static linking.

### Example
```kv
rust_block {
    fn compute_pi(n: i32) -> f64 {
        3.14159
    }
}
```

### Mechanism
1.  Creates a temporary Cargo workspace.
2.  Preps `src/lib.rs` with `#[no_mangle]` and `pub extern "C"`.
3.  Executes `cargo build --release`.
4.  Links `libvolt_rust.a` into the final executable.

---

## 🟡 JavaScript Blocks (`js_block`)

JavaScript is powered by an **embedded QuickJS engine**, ensuring zero external dependencies.

### Example
```kv
js_block {
    console.log("Hello from QuickJS!");
    const x = 10 + 20;
}
```

### Mechanism
*   The QuickJS source code is integrated into the KS-Volt workspace (`ks-volt/deps/`).
*   The captured JS source is baked into the binary as a constant string.
*   At runtime, a micro-context is initialized via `JS_NewRuntime()` and scripts are evaluated instantly.

---

## 🐍 Python Blocks (`py_block`)

Python integration uses the standard CPython development headers.

### Example
```kv
py_block {
    import math
    print(f"Python math: {math.sqrt(144)}")
}
```

### Mechanism
*   Includes `<Python.h>` in the generated C code.
*   The compiler automatically detects Python development flags using `python3-config`.
*   Executes via `PyRun_SimpleString` with a persistent initialization state.
