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

## 📂 Automatic UI Loader (`import_ui`)

The `import_ui` directive enables a file-system-based routing and component registration system, balancing automated layout loading with backend control.

### Usage
```kv
import_ui "web"
```

### Mechanism
- **Global Components**: All `.kv` files inside the `web/components/` directory are recursively scanned and registered as global building blocks.
- **Automated Routing**: Files inside `web/pages/` are automatically mapped to static URL paths.
    - `web/pages/index.kv` maps to `/`
    - `web/pages/about.kv` maps to `/about`
    - `web/pages/contact.kv` maps to `/contact`

## 🌐 Web Routing Engine

The `web_block` provides a dedicated DSL for defining high-performance controllers and routing logic.

### DSL Structure
```kv
web_block "main_app" {
    before_each -> {
        print("Incoming request...")
    }

    path("/login") -> {
        // Explicit POST handler
        if (request_header("Method") == "POST") {
            // ... auth logic
        }
    }

    path_ws("/dashboard") -> {
        // Real-time metrics via WebSocket
    }
}
```

### Route Precedence
Explicit `path()` and `path_ws()` blocks defined inside a `web_block` always take precedence over automatically generated routes from `import_ui`. This allows developers to intercept static pages and inject dynamic data or handle complex state mutations seamlessly.

## ⚡ Zero-Allocation & Backpressure

*   **Zero-Allocation Rendering**: When a component is called, it appends data directly to a `VoltBuffer`, ensuring minimal RAM usage.
*   **Built-in Backpressure**: The `VoltBuffer` API integrates with the KS-Volt Netpoller. If a slow client causes kernel write buffers to saturate, the generating green thread is automatically parked until the buffer drains, preventing memory exhaustion under high load.
