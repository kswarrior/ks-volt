# 🎨 KS-Volt Frontend & Components

KS-Volt revolutionizes frontend development with **Hyper-Static Components** and an integrated high-velocity web routing DSL.

## 🏗️ Hyper-Static Components

Components in KS-Volt are not interpreted at runtime. Instead, the compiler performs **Build-Time Ingestion**, translating your `.kv` components into optimized C rendering functions that use the zero-allocation `VoltBuffer` API.

### Definition
```kv
component Header(title) {
    `<h1>${title}</h1>`
}
```

### Import & Aliasing
KS-Volt supports namespaced imports to prevent symbol collisions in large projects:
```kv
import_component "ui/navbar.kv" as UI
UI.Navbar("Home")
```

## 🌐 Web Routing Engine

The `web_block` provides a dedicated DSL for defining high-performance controllers and routing logic.

### DSL Structure
```kv
web_block "api_v1" {
    before_each -> {
        print("Incoming request...")
    }

    path("/status") -> {
        `{"status": "online"}`
    }

    path_ws("/events") -> {
        // WebSocket handler logic
    }
}
```

### Internal Implementation
*   **Routing Jump Tables**: Routes are registered using C constructors (`__attribute__((constructor))`), creating an efficient jump table for the dispatcher.
*   **Buffer Inlining**: Interpolated string literals inside routes are automatically directed to the response buffer, avoiding intermediate heap allocations.

## ⚡ Zero-Allocation Rendering

When a component is called within a web route or another component, it receives a reference to a `VoltBuffer`. The rendering process consists of direct character-array appends, ensuring minimal RAM usage even under high concurrency.
