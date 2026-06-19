# KS-Volt Language Reference

A complete list of all keywords, directives, and built-in primitives in KS-Volt.

## Keywords & Control Flow
| Keyword | Description |
| :--- | :--- |
| `if` | Conditional execution |
| `else` | Alternative execution path |
| `true` | Boolean true |
| `false` | Boolean false |
| `loop` | Iterate over an array or map |
| `in` | Membership operator for loops |
| `try` | Start an exception guardrail block |
| `catch` | Trap OS/runtime exceptions |
| `fn` | Define a first-class function |
| `return` | Return a value from a function |

## Web Platform Directives (inside `web_block`)
| Directive | Description |
| :--- | :--- |
| `web_block "name"` | Define an HTTP server instance |
| `before_each` | Global middleware executed before every route |
| `path("/url")` | Define an HTTP GET/POST route |
| `path_ws("/url")` | Define a persistent WebSocket route |
| `render_fragment()` | Stream raw HTML bypassing global layouts/headers |
| `dispatch_job() -> {}` | Offload a block to the M:N scheduler from a route |
| `request_header(name)` | Native extraction of HTTP headers |
| `redirect(url)` | Immediate 302 redirect |
| `json(obj)` | Serialize object to JSON and set application/json mime |
| `request_form(key)` | Extract value from POST form payload |
| `request_json()` | Parse request body as a Volt Map |

## Concurrency Primitives
| Command | Description |
| :--- | :--- |
| `spawn { ... }` | Execute block in a new green thread |
| `interval(ms) { ... }` | Execute block repeatedly every N milliseconds |
| `on("event") { ... }` | Register a listener for a global event |
| `emit("event", data)` | Trigger global event and wake listeners |
| `sleep(sec)` | Suspend current thread for N seconds |

## File System Macros (Direct POSIX)
| Macro | POSIX Mapping | Description |
| :--- | :--- | :--- |
| `fs_rm(path)` | `unlink` | Delete a file |
| `fs_mv(src, dst)` | `rename` | Rename or move a file |
| `fs_cp(src, dst)` | `fread/fwrite` | Streaming buffer copy |
| `fs_touch(path)` | `fopen(a)` | Create file or update timestamp |
| `fs_cat(path)` | `fread/fwrite` | Stream file content to stdout |

## Built-in Primitives
| Function | Description |
| :--- | :--- |
| `print(value)` | Write value to stdout |
| `json_parse(str)` | Convert JSON string to Volt Map/Array |
| `db_save(k, v)` | Save key-value pair to `volt_db.json` (thread-safe) |
| `db_get(k)` | Retrieve value from persistence layer |
| `file_write(f, d)` | Atomic write of data to file path |
| `get_addr(val)` | Return the memory address string of a value |
| `exit(code)` | Immediately terminate the process |

## String Methods
| Method | Description |
| :--- | :--- |
| `.trim()` | Remove leading/trailing whitespace |
| `.upper()` | Convert string to uppercase |

## Component System
| Directive | Description |
| :--- | :--- |
| `component Name(params)` | Define a hyper-static UI component |
| `import_component "path"` | Ingest a KV file as a namespaced component module |
| `as Alias` | Assign a namespace to an imported component module |

## Polyglot Blocks
| Block Type | Description |
| :--- | :--- |
| `go_block { ... }` | Embed Go source (compiled via c-archive) |
| `rust_block { ... }` | Embed Rust source (compiled via staticlib) |
| `js_block { ... }` | Embed JavaScript (executed via QuickJS) |
| `py_block { ... }` | Embed Python source (C-API embedding) |
