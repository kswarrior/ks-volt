package compiler

import (
	"fmt"
	"ks-volt/ast"
	"ks-volt/lexer"
	"ks-volt/parser"
	"ks-volt/token"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Compiler struct {
	globalVars       map[string]bool
	components       map[string]bool
	funcID           int
	LinkLibs         []string
	PythonNeeded     bool
	ComponentAliases map[string]string
	curComponent     string
	curPrefix        string
	webBlocks        []string
	importedFiles    map[string]bool
}

func New() *Compiler {
	return &Compiler{
		globalVars:       make(map[string]bool),
		components:       make(map[string]bool),
		ComponentAliases: make(map[string]string),
		importedFiles:    make(map[string]bool),
	}
}

func (c *Compiler) Compile(program *ast.Program) string {
	var sb strings.Builder

	// Definitive C Header, GMP Scheduler, and Unmanaged Runtime
	sb.WriteString(`#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <pthread.h>
#include <unistd.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <sys/time.h>
#include <fcntl.h>
#include <errno.h>
#include <malloc.h>
#include <setjmp.h>
#include <stdbool.h>
#include <ctype.h>
#include <sys/stat.h>
#include <dirent.h>

#define MAX_TASKS 8192
#define NUM_WORKERS 4

typedef enum { TYPE_STR, TYPE_INT, TYPE_BOOL, TYPE_ARRAY, TYPE_MAP, TYPE_FN } VoltType;

struct VoltValue;
typedef struct VoltValue* (*VoltFn)(int, struct VoltValue**);

typedef struct VoltBuffer {
    char* data;
    size_t len;
    size_t cap;
} VoltBuffer;

VoltBuffer* volt_buf_new() {
    VoltBuffer* b = malloc(sizeof(VoltBuffer));
    b->cap = 4096; b->len = 0; b->data = malloc(b->cap);
    b->data[0] = '\0';
    return b;
}

void volt_buf_append(VoltBuffer* b, const char* s) {
    if (!b || !s) return;
    size_t slen = strlen(s);
    if (b->len + slen + 1 > b->cap) {
        b->cap = (b->len + slen + 4096) * 2;
        b->data = realloc(b->data, b->cap);
    }
    strcpy(b->data + b->len, s); b->len += slen;
}

void volt_buf_free(VoltBuffer* b) {
    if (b) { free(b->data); free(b); }
}

void volt_value_free(struct VoltValue* v);
struct VoltValue* volt_value_copy(struct VoltValue* v);

typedef struct VoltValue {
    VoltType type;
    union {
        char* s;
        long long i;
        bool b;
        struct { struct VoltValue** elements; int len; } a;
        struct { char** keys; struct VoltValue** values; int len; } m;
        VoltFn f;
    };
} VoltValue;

typedef struct Task {
    void (*func)(void*);
    void* arg;
} Task;

typedef struct {
    Task queue[MAX_TASKS];
    int head, tail;
    pthread_mutex_t lock;
} Processor;

Processor processors[NUM_WORKERS];
pthread_t workers[NUM_WORKERS];
__thread int worker_id;
__thread jmp_buf* current_jmp_env;
pthread_mutex_t db_file_lock = PTHREAD_MUTEX_INITIALIZER;

void schedule_task(int p_id, void (*func)(void*), void* arg) {
    Processor* p = &processors[p_id];
    pthread_mutex_lock(&p->lock);
    int next = (p->tail + 1) % MAX_TASKS;
    if (next != p->head) {
        p->queue[p->tail].func = func;
        p->queue[p->tail].arg = arg;
        p->tail = next;
    }
    pthread_mutex_unlock(&p->lock);
}

void* worker_loop(void* arg) {
    worker_id = *(int*)arg;
    while (1) {
        Processor* p = &processors[worker_id];
        Task t = {NULL, NULL};
        pthread_mutex_lock(&p->lock);
        if (p->head != p->tail) {
            t = p->queue[p->head];
            p->head = (p->head + 1) % MAX_TASKS;
        }
        pthread_mutex_unlock(&p->lock);
        if (t.func) {
            t.func(t.arg);
            malloc_trim(0);
        } else {
            int target = rand() % NUM_WORKERS;
            if (target != worker_id) {
                pthread_mutex_lock(&processors[target].lock);
                if (processors[target].head != processors[target].tail) {
                    t = processors[target].queue[processors[target].head];
                    processors[target].head = (processors[target].head + 1) % MAX_TASKS;
                }
                pthread_mutex_unlock(&processors[target].lock);
                if (t.func) t.func(t.arg);
            }
            usleep(1000);
        }
    }
    return NULL;
}

// Runtime Primitives
VoltValue* make_str(const char* s) {
    VoltValue* v = malloc(sizeof(VoltValue));
    v->type = TYPE_STR; v->s = strdup(s ? s : ""); return v;
}
VoltValue* make_int(long long i) {
    VoltValue* v = malloc(sizeof(VoltValue));
    v->type = TYPE_INT; v->i = i; return v;
}
VoltValue* make_bool(bool b) {
    VoltValue* v = malloc(sizeof(VoltValue));
    v->type = TYPE_BOOL; v->b = b; return v;
}
VoltValue* make_fn(VoltFn f) {
    VoltValue* v = malloc(sizeof(VoltValue));
    v->type = TYPE_FN; v->f = f; return v;
}

VoltValue* volt_value_copy(VoltValue* v) {
    if (!v) return NULL;
    VoltValue* res = malloc(sizeof(VoltValue));
    res->type = v->type;
    switch(v->type) {
        case TYPE_STR: res->s = strdup(v->s); break;
        case TYPE_INT: res->i = v->i; break;
        case TYPE_BOOL: res->b = v->b; break;
        case TYPE_FN: res->f = v->f; break;
        case TYPE_ARRAY:
            res->a.len = v->a.len;
            res->a.elements = malloc(v->a.len * sizeof(VoltValue*));
            for(int i=0; i<v->a.len; i++) res->a.elements[i] = volt_value_copy(v->a.elements[i]);
            break;
        case TYPE_MAP:
            res->m.len = v->m.len;
            res->m.keys = malloc(v->m.len * sizeof(char*));
            res->m.values = malloc(v->m.len * sizeof(VoltValue*));
            for(int i=0; i<v->m.len; i++) {
                res->m.keys[i] = strdup(v->m.keys[i]);
                res->m.values[i] = volt_value_copy(v->m.values[i]);
            }
            break;
    }
    return res;
}

const char* to_str_buf(VoltValue* v, char* buf) {
    if (!v) return "";
    if (v->type == TYPE_STR) return v->s;
    if (v->type == TYPE_INT) sprintf(buf, "%lld", v->i);
    else if (v->type == TYPE_BOOL) sprintf(buf, "%s", v->b ? "true" : "false");
    else if (v->type == TYPE_FN) sprintf(buf, "[function]");
    else return "complex";
    return buf;
}

void volt_buf_append_value(VoltBuffer* b, VoltValue* v) {
    char buf[128];
    volt_buf_append(b, to_str_buf(v, buf));
}

VoltValue* dynamic_add(VoltValue* a, VoltValue* b) {
    char buf1[128], buf2[128];
    const char* s1 = to_str_buf(a, buf1); const char* s2 = to_str_buf(b, buf2);
    char* res = malloc(strlen(s1) + strlen(s2) + 1);
    strcpy(res, s1); strcat(res, s2);
    VoltValue* rv = make_str(res); free(res);
    volt_value_free(a); volt_value_free(b);
    return rv;
}

VoltValue* str_trim(VoltValue* v) {
    if (!v || v->type != TYPE_STR) { volt_value_free(v); return make_str(""); }
    char* s = v->s;
    while(isspace(*s)) s++;
    if(*s == 0) { volt_value_free(v); return make_str(""); }
    char* end = s + strlen(s) - 1;
    while(end > s && isspace(*end)) end--;
    char* res = strndup(s, end - s + 1);
    VoltValue* rv = make_str(res); free(res);
    volt_value_free(v);
    return rv;
}

VoltValue* str_upper(VoltValue* v) {
    if (!v || v->type != TYPE_STR) { volt_value_free(v); return make_str(""); }
    char* s = strdup(v->s);
    for(int i=0; s[i]; i++) s[i] = toupper(s[i]);
    VoltValue* res = make_str(s); free(s);
    volt_value_free(v);
    return res;
}

VoltValue* json_parse(const char* json) {
    VoltValue* m = malloc(sizeof(VoltValue));
    m->type = TYPE_MAP; m->m.len = 0;
    m->m.keys = malloc(20 * sizeof(char*));
    m->m.values = malloc(20 * sizeof(VoltValue*));
    char* s = strdup(json);
    char* p = strtok(s, "{}\",: ");
    while(p && m->m.len < 20) {
        m->m.keys[m->m.len] = strdup(p);
        p = strtok(NULL, "{}\",: ");
        if (p) {
            if (isdigit(*p)) m->m.values[m->m.len++] = make_int(atoll(p));
            else m->m.values[m->m.len++] = make_str(p);
        }
        p = strtok(NULL, "{}\",: ");
    }
    free(s);
    return m;
}

VoltValue* map_get(VoltValue* m, const char* key) {
    if (m->type != TYPE_MAP) return make_str("");
    for(int i=0; i<m->m.len; i++) {
        if (strcmp(m->m.keys[i], key) == 0) return m->m.values[i];
    }
    return make_str("");
}

void db_save(const char* key, const char* val) {
    pthread_mutex_lock(&db_file_lock);
    FILE* f = fopen("volt_db.json", "a");
    if(f) { fprintf(f, "%s:%s\n", key, val); fclose(f); }
    pthread_mutex_unlock(&db_file_lock);
}

VoltValue* db_get(const char* key) { return make_str("ready"); }

void volt_file_write(const char* fn, const char* data) {
    pthread_mutex_lock(&db_file_lock);
    FILE* f = fopen(fn, "w");
    if (!f) { pthread_mutex_unlock(&db_file_lock); if (current_jmp_env) longjmp(*current_jmp_env, 1); return; }
    fputs(data, f); fclose(f);
    pthread_mutex_unlock(&db_file_lock);
}

void volt_value_free(VoltValue* v) {
    if (!v) return;
    if (v->type == TYPE_STR) free(v->s);
    else if (v->type == TYPE_ARRAY) {
        for(int i=0; i<v->a.len; i++) volt_value_free(v->a.elements[i]);
        free(v->a.elements);
    } else if (v->type == TYPE_MAP) {
        for(int i=0; i<v->m.len; i++) {
            free(v->m.keys[i]);
            volt_value_free(v->m.values[i]);
        }
        free(v->m.keys);
        free(v->m.values);
    }
    free(v);
}

void connect_bot(const char* ip, int port, void (*cb)(void*)) {
    int sock = socket(AF_INET, SOCK_STREAM, 0);
    struct sockaddr_in addr = {0};
    addr.sin_family = AF_INET;
    addr.sin_port = htons(port);
    inet_pton(AF_INET, ip, &addr.sin_addr);
    struct timeval tv = {2, 0};
    setsockopt(sock, SOL_SOCKET, SO_SNDTIMEO, (const char*)&tv, sizeof(tv));
    if (connect(sock, (struct sockaddr*)&addr, sizeof(addr)) == 0) {
        cb(NULL);
    } else {
        cb(NULL);
    }
    close(sock);
}

typedef struct { int ms; void (*func)(void*); } IntervalArg;
void* interval_runner(void* arg) {
    IntervalArg* ia = (IntervalArg*)arg;
    while(1) { usleep(ia->ms * 1000); schedule_task(rand() % NUM_WORKERS, ia->func, NULL); }
}
void start_interval(int ms, void (*func)(void*)) {
    IntervalArg* arg = malloc(sizeof(IntervalArg)); arg->ms = ms; arg->func = func;
    pthread_t t; pthread_create(&t, NULL, interval_runner, arg);
}

// FS Shortcuts
void fs_rm(const char* path) { unlink(path); }
void fs_mv(const char* src, const char* dst) { rename(src, dst); }
void fs_touch(const char* path) { FILE* f = fopen(path, "a"); if(f) fclose(f); }
void fs_cat(const char* path) {
    FILE* f = fopen(path, "r");
    if(!f) return;
    char buf[4096];
    size_t n;
    while((n = fread(buf, 1, sizeof(buf), f)) > 0) fwrite(buf, 1, n, stdout);
    fclose(f);
}
void fs_cp(const char* src, const char* dst) {
    FILE* s = fopen(src, "rb");
    if(!s) return;
    FILE* d = fopen(dst, "wb");
    if(!d) { fclose(s); return; }
    char buf[8192];
    size_t n;
    while((n = fread(buf, 1, sizeof(buf), s)) > 0) fwrite(buf, 1, n, d);
    fclose(s); fclose(d);
}

// Routing Logic
typedef struct Route { char* path; void (*handler)(void*); bool is_ws; } Route;
typedef struct Router { char* name; Route routes[100]; int count; void (*before)(void*); } Router;

void volt_start_web_server(Router* r, int port) {
    printf("Web server '%s' started on port %d with %d routes\n", r->name, port, r->count);
}

void volt_set_value(VoltValue** dest, VoltValue* src) {
    if (*dest == src) return;
    if (*dest) volt_value_free(*dest);
    *dest = src;
}

const char* to_str(VoltValue* v) {
    static __thread char b[4][128];
    static __thread int i = 0;
    i = (i + 1) % 4;
    return to_str_buf(v, b[i]);
}
`)

	c.collectGlobalVars(program)
	for v := range c.globalVars {
		sb.WriteString("VoltValue* " + v + ";\n")
	}

	var funcs strings.Builder
	var mainBody strings.Builder
	for _, stmt := range program.Statements {
		mainBody.WriteString(c.transpileStatement(stmt, &funcs, "    "))
	}

	if c.PythonNeeded {
		sb.WriteString("#include <Python.h>\n")
	}
	sb.WriteString("#include \"deps/quickjs.h\"\n")
	sb.WriteString("#include \"deps/quickjs-libc.h\"\n")

	sb.WriteString(funcs.String())
	sb.WriteString("\nint main(int argc, char** argv) {\n")
	sb.WriteString("    srand(time(NULL));\n")
	if c.PythonNeeded {
		sb.WriteString("    Py_Initialize();\n")
	}
	sb.WriteString("    for(int i=0; i<NUM_WORKERS; i++) {\n")
	sb.WriteString("        int* id = malloc(sizeof(int)); *id = i;\n")
	sb.WriteString("        pthread_mutex_init(&processors[i].lock, NULL);\n")
	sb.WriteString("        pthread_create(&workers[i], NULL, worker_loop, id);\n")
	sb.WriteString("    }\n")
	sb.WriteString(mainBody.String())
	if c.PythonNeeded {
		sb.WriteString("    Py_Finalize();\n")
	}
	sb.WriteString("    while(1) sleep(1); return 0;\n}\n")

	return sb.String()
}

func (c *Compiler) collectGlobalVars(program *ast.Program) {
	var walker func(node interface{})
	walker = func(node interface{}) {
		if node == nil {
			return
		}
		switch n := node.(type) {
		case *ast.Program:
			for _, s := range n.Statements {
				walker(s)
			}
		case *ast.AssignmentStatement:
			c.globalVars[n.Name.Value] = true
			walker(n.Value)
		case *ast.FunctionStatement:
			c.globalVars[n.Name.Value] = true
			walker(n.Body)
		case *ast.BlockStatement:
			for _, s := range n.Statements {
				walker(s)
			}
		case *ast.LoopStatement:
			c.globalVars[n.Variable.Value] = true
			walker(n.Body)
		case *ast.TryCatchStatement:
			walker(n.TryBody)
			c.globalVars[n.CatchVariable.Value] = true
			walker(n.CatchBody)
		case *ast.IfStatement:
			walker(n.Consequence)
			if n.Alternative != nil {
				walker(n.Alternative)
			}
		case *ast.SpawnStatement:
			walker(n.Body)
		case *ast.InfixExpression:
			walker(n.Left)
			walker(n.Right)
		case *ast.CallExpression:
			walker(n.Function)
			for _, a := range n.Arguments {
				walker(a)
			}
		case *ast.IndexExpression:
			walker(n.Left)
			walker(n.Index)
		case *ast.ArrayLiteral:
			for _, e := range n.Elements {
				walker(e)
			}
		case *ast.ExpressionStatement:
			walker(n.Expression)
		case *ast.MethodCallExpression:
			walker(n.Object)
			for _, a := range n.Arguments {
				walker(a)
			}
		}
	}
	walker(program)
}

func (c *Compiler) transpileStatement(stmt ast.Statement, funcs *strings.Builder, indent string) string {
	if stmt == nil {
		return ""
	}
	switch s := stmt.(type) {
	case *ast.FSMacroStatement:
		res := indent + "{\n"
		cArgs := []string{}
		temps := []string{}
		for i, arg := range s.Args {
			code, isTemp := c.transpileExpression(arg)
			vName := fmt.Sprintf("_fa%d", i)
			res += fmt.Sprintf("%s    VoltValue* %s = %s;\n", indent, vName, code)
			cArgs = append(cArgs, "to_str("+vName+")")
			if isTemp {
				temps = append(temps, vName)
			}
		}
		switch s.Token.Type {
		case token.FS_RM:
			res += indent + "    fs_rm(" + cArgs[0] + ");\n"
		case token.FS_MV:
			res += indent + "    fs_mv(" + cArgs[0] + ", " + cArgs[1] + ");\n"
		case token.FS_CP:
			res += indent + "    fs_cp(" + cArgs[0] + ", " + cArgs[1] + ");\n"
		case token.FS_TOUCH:
			res += indent + "    fs_touch(" + cArgs[0] + ");\n"
		case token.FS_CAT:
			res += indent + "    fs_cat(" + cArgs[0] + ");\n"
		}
		for _, t := range temps {
			res += indent + "    volt_value_free(" + t + ");\n"
		}
		res += indent + "}\n"
		return res
	case *ast.PolyglotBlockStatement:
		switch s.Token.Type {
		case token.PY_BLOCK:
			c.PythonNeeded = true
			code := strings.ReplaceAll(s.Code, "\\", "\\\\")
			code = strings.ReplaceAll(code, "\"", "\\\"")
			code = strings.ReplaceAll(code, "\n", "\\n")
			return indent + "{\n" +
				indent + "    PyRun_SimpleString(\"" + code + "\");\n" +
				indent + "}\n"
		case token.JS_BLOCK:
			code := strings.ReplaceAll(s.Code, "\\", "\\\\")
			code = strings.ReplaceAll(code, "\"", "\\\"")
			code = strings.ReplaceAll(code, "\n", "\\n")
			return indent + "{\n" +
				indent + "    JSRuntime *rt = JS_NewRuntime();\n" +
				indent + "    JSContext *ctx = JS_NewContext(rt);\n" +
				indent + "    js_std_add_helpers(ctx, 0, NULL);\n" +
				indent + "    const char *js_code = \"" + code + "\";\n" +
				indent + "    JS_Eval(ctx, js_code, strlen(js_code), \"volt.js\", JS_EVAL_TYPE_GLOBAL);\n" +
				indent + "    JS_FreeContext(ctx);\n" +
				indent + "    JS_FreeRuntime(rt);\n" +
				indent + "}\n"
		case token.GO_BLOCK:
			lines := strings.Split(s.Code, "\n")
			var wrappedLines []string
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "func ") {
					parts := strings.Split(strings.TrimPrefix(trimmed, "func "), "(")
					if len(parts) > 0 {
						funcName := strings.TrimSpace(parts[0])
						wrappedLines = append(wrappedLines, "//export "+funcName)
					}
				}
				wrappedLines = append(wrappedLines, line)
			}
			goCode := "package main\nimport \"C\"\n" + strings.Join(wrappedLines, "\n") + "\nfunc main() {}\n"
			os.WriteFile("volt_bridge.go", []byte(goCode), 0644)
			cmd := exec.Command("go", "build", "-buildmode=c-archive", "-o", "volt_go.a", "volt_bridge.go")
			if err := cmd.Run(); err != nil {
				fmt.Printf("Go block compilation error: %v\n", err)
			} else {
				c.LinkLibs = append(c.LinkLibs, "volt_go.a")
			}
			return indent + "// Go Block compiled and linked\n"
		case token.RUST_BLOCK:
			// Auto prepend attributes
			lines := strings.Split(s.Code, "\n")
			var rustBody []string
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "fn ") || strings.HasPrefix(trimmed, "pub fn ") {
					rustBody = append(rustBody, "#[no_mangle]")
					if !strings.HasPrefix(trimmed, "pub ") {
						line = strings.Replace(line, "fn ", "pub extern \"C\" fn ", 1)
					} else {
						line = strings.Replace(line, "pub fn ", "pub extern \"C\" fn ", 1)
					}
				}
				rustBody = append(rustBody, line)
			}
			rustCode := "#![allow(dead_code)]\n" + strings.Join(rustBody, "\n")
			os.MkdirAll("volt_rust/src", 0755)
			os.WriteFile("volt_rust/src/lib.rs", []byte(rustCode), 0644)
			cargoToml := "[package]\nname = \"volt_rust\"\nversion = \"0.1.0\"\nedition = \"2021\"\n[lib]\ncrate-type = [\"staticlib\"]\n"
			os.WriteFile("volt_rust/Cargo.toml", []byte(cargoToml), 0644)
			cmd := exec.Command("cargo", "build", "--release", "--manifest-path", "volt_rust/Cargo.toml")
			if err := cmd.Run(); err != nil {
				fmt.Printf("Rust block compilation error: %v\n", err)
			} else {
				c.LinkLibs = append(c.LinkLibs, "volt_rust/target/release/libvolt_rust.a")
			}
			return indent + "// Rust Block compiled and linked\n"
		}
		return ""
	case *ast.ComponentDefinition:
		c.components[s.Name.Value] = true
		oldComp := c.curComponent
		c.curComponent = s.Name.Value
		fName := "render_" + s.Name.Value
		funcs.WriteString("void " + fName + "(VoltBuffer* ctx, int argc, VoltValue** argv) {\n")
		for i, p := range s.Parameters {
			funcs.WriteString("    VoltValue* " + p.Value + " = argv[" + strconv.Itoa(i) + "];\n")
		}
		for _, bs := range s.Body.Statements {
			funcs.WriteString(c.transpileStatement(bs, funcs, "    "))
		}
		funcs.WriteString("}\n")
		c.curComponent = oldComp
		return ""
	case *ast.ImportComponentStatement:
		c.processImport(s.Path, s.Alias.Value, funcs)
		return ""
	case *ast.WebBlockStatement:
		name := s.Name
		if name == "" {
			name = "main_daemon"
		}
		routerName := "router_" + name
		funcs.WriteString("Router " + routerName + " = { .name = \"" + name + "\", .count = 0 };\n")
		for _, bs := range s.Body.Statements {
			switch r := bs.(type) {
			case *ast.PathStatement:
				handlerName := "handler_" + strconv.Itoa(c.funcID)
				c.funcID++
				funcs.WriteString("void " + handlerName + "(void* arg) {\n")
				funcs.WriteString(c.transpileStatement(r.Body, funcs, "    "))
				funcs.WriteString("}\n")
				funcs.WriteString("__attribute__((constructor)) void init_" + handlerName + "() { " +
					routerName + ".routes[" + routerName + ".count++] = (Route){\"" + r.Path + "\", " + handlerName + ", false }; }\n")
			case *ast.PathWsStatement:
				handlerName := "handler_" + strconv.Itoa(c.funcID)
				c.funcID++
				funcs.WriteString("void " + handlerName + "(void* arg) {\n")
				funcs.WriteString(c.transpileStatement(r.Body, funcs, "    "))
				funcs.WriteString("}\n")
				funcs.WriteString("__attribute__((constructor)) void init_" + handlerName + "() { " +
					routerName + ".routes[" + routerName + ".count++] = (Route){\"" + r.Path + "\", " + handlerName + ", true }; }\n")
			case *ast.BeforeEachStatement:
				handlerName := "before_" + name + "_" + strconv.Itoa(c.funcID)
				c.funcID++
				funcs.WriteString("void " + handlerName + "(void* arg) {\n")
				funcs.WriteString(c.transpileStatement(r.Body, funcs, "    "))
				funcs.WriteString("}\n")
				funcs.WriteString("__attribute__((constructor)) void init_" + handlerName + "() { " + routerName + ".before = " + handlerName + "; }\n")
			}
		}
		return indent + "volt_start_web_server(&" + routerName + ", 8080);\n"
	case *ast.AssignmentStatement:
		code, isTemp := c.transpileExpression(s.Value)
		if isTemp {
			return indent + "volt_set_value(&" + s.Name.Value + ", " + code + ");\n"
		}
		return indent + "volt_set_value(&" + s.Name.Value + ", volt_value_copy(" + code + "));\n"
	case *ast.FunctionStatement:
		fName := "volt_fn_" + s.Name.Value
		funcs.WriteString("VoltValue* " + fName + "_impl(int argc, VoltValue** argv) {\n")
		for i, p := range s.Parameters {
			funcs.WriteString("    VoltValue* " + p.Value + " = argv[" + strconv.Itoa(i) + "];\n")
		}
		for _, bs := range s.Body.Statements {
			funcs.WriteString(c.transpileStatement(bs, funcs, "    "))
		}
		funcs.WriteString("    return make_int(0);\n}\n")
		return indent + s.Name.Value + " = make_fn(" + fName + "_impl);\n"
	case *ast.ReturnStatement:
		code, isTemp := c.transpileExpression(s.ReturnValue)
		if isTemp {
			return indent + "return " + code + ";\n"
		}
		return indent + "return volt_value_copy(" + code + ");\n"
	case *ast.IfStatement:
		code, isTemp := c.transpileExpression(s.Condition)
		res := indent + "{\n"
		res += indent + "    VoltValue* _cond = " + code + ";\n"
		res += indent + "    bool _b = _cond->b;\n"
		if isTemp {
			res += indent + "    volt_value_free(_cond);\n"
		}
		res += indent + "    if (_b) {\n"
		for _, st := range s.Consequence.Statements {
			res += c.transpileStatement(st, funcs, indent+"        ")
		}
		res += indent + "    }"
		if s.Alternative != nil {
			res += " else {\n"
			for _, st := range s.Alternative.Statements {
				res += c.transpileStatement(st, funcs, indent+"        ")
			}
			res += indent + "    }"
		}
		res += "\n" + indent + "}\n"
		return res
	case *ast.ExpressionStatement:
		if c.curComponent != "" {
			if is, ok := s.Expression.(*ast.InterpolatedStringLiteral); ok {
				return c.transpileToBuffer(is, "ctx", indent)
			}
			if call, ok := s.Expression.(*ast.CallExpression); ok {
				if ident, ok := call.Function.(*ast.Identifier); ok && ident.Value == "print" {
					if is, ok := call.Arguments[0].(*ast.InterpolatedStringLiteral); ok {
						return c.transpileToBuffer(is, "ctx", indent) + indent + "volt_buf_append(ctx, \"\\n\");\n"
					}
				}
			}
		}
		code, isTemp := c.transpileExpression(s.Expression)
		if isTemp {
			return indent + "volt_value_free(" + code + ");\n"
		}
		return indent + code + ";\n"
	case *ast.LoopStatement:
		code, isTemp := c.transpileExpression(s.Iterable)
		var body strings.Builder
		for _, bs := range s.Body.Statements {
			body.WriteString(c.transpileStatement(bs, funcs, indent+"    "))
		}
		res := indent + "{ VoltValue* _it = " + code + ";\n"
		res += indent + "    if (_it->type == TYPE_ARRAY) {\n"
		res += indent + "        for(int _idx = 0; _idx < _it->a.len; _idx++) {\n"
		res += indent + "            volt_set_value(&" + s.Variable.Value + ", volt_value_copy(_it->a.elements[_idx]));\n"
		res += body.String()
		res += indent + "        }\n"
		res += indent + "    }\n"
		if isTemp {
			res += indent + "    volt_value_free(_it);\n"
		}
		res += indent + "}\n"
		return res
	case *ast.TryCatchStatement:
		id := strconv.Itoa(c.funcID)
		c.funcID++
		funcs.WriteString("void volt_try_" + id + "(void* arg) {\n")
		for _, ts := range s.TryBody.Statements {
			funcs.WriteString(c.transpileStatement(ts, funcs, "    "))
		}
		funcs.WriteString("}\n")
		return indent + "{ jmp_buf env_" + id + "; current_jmp_env = &env_" + id + ";\n" +
			indent + "if (setjmp(env_" + id + ") == 0) {\n" +
			indent + "    volt_try_" + id + "(NULL);\n" +
			indent + "} else {\n" +
			indent + "    volt_set_value(&" + s.CatchVariable.Value + ", make_str(\"OS Exception\"));\n" +
			c.transpileStatement(s.CatchBody, funcs, indent+"    ") + indent + "} }\n"
	case *ast.SpawnStatement:
		id := strconv.Itoa(c.funcID)
		c.funcID++
		funcs.WriteString("void volt_func_" + id + "(void* arg) {\n")
		for _, bs := range s.Body.Statements {
			funcs.WriteString(c.transpileStatement(bs, funcs, "    "))
		}
		funcs.WriteString("}\n")
		if s.Name.Value == "connect_bot" {
			code0, isTemp0 := c.transpileExpression(s.Args[0])
			code1, isTemp1 := c.transpileExpression(s.Args[1])
			res := indent + "{\n"
			res += indent + "    VoltValue* _a0 = " + code0 + ";\n"
			res += indent + "    VoltValue* _a1 = " + code1 + ";\n"
			res += indent + "    connect_bot(to_str(_a0), (int)_a1->i, volt_func_" + id + ");\n"
			if isTemp0 {
				res += indent + "    volt_value_free(_a0);\n"
			}
			if isTemp1 {
				res += indent + "    volt_value_free(_a1);\n"
			}
			res += indent + "}\n"
			return res
		}
		if s.Name.Value == "interval" {
			code, isTemp := c.transpileExpression(s.Args[0])
			res := indent + "{\n"
			res += indent + "    VoltValue* _a = " + code + ";\n"
			res += indent + "    start_interval((int)_a->i, volt_func_" + id + ");\n"
			if isTemp {
				res += indent + "    volt_value_free(_a);\n"
			}
			res += indent + "}\n"
			return res
		}
		return indent + "schedule_task(rand() % NUM_WORKERS, volt_func_" + id + ", NULL);\n"
	case *ast.BeforeEachStatement:
		return indent + "// Middleware block execution\n"
	}
	return ""
}

func (c *Compiler) processImport(path string, aliasPrefix string, funcs *strings.Builder) {
	if c.importedFiles[path] {
		return
	}
	c.importedFiles[path] = true

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Error importing component %s: %v\n", path, err)
		return
	}

	subL := lexer.New(string(data))
	subP := parser.New(subL)
	subProg := subP.ParseProgram()

	oldPrefix := c.curPrefix
	c.curPrefix = aliasPrefix

	for _, subStmt := range subProg.Statements {
		switch st := subStmt.(type) {
		case *ast.ComponentDefinition:
			originalName := st.Name.Value
			fullName := aliasPrefix + originalName
			st.Name.Value = fullName
			c.components[fullName] = true
			c.ComponentAliases[fullName] = st.Name.Value
			c.transpileStatement(st, funcs, "")
			st.Name.Value = originalName
		case *ast.ImportComponentStatement:
			c.processImport(st.Path, aliasPrefix+st.Alias.Value, funcs)
		}
	}
	c.curPrefix = oldPrefix
}

func (c *Compiler) transpileToBuffer(expr ast.Expression, bufName string, indent string) string {
	if is, ok := expr.(*ast.InterpolatedStringLiteral); ok {
		res := ""
		for _, seg := range is.Segments {
			if sl, ok := seg.(*ast.StringLiteral); ok {
				esc := strings.ReplaceAll(sl.Value, "\\", "\\\\")
				esc = strings.ReplaceAll(esc, "\"", "\\\"")
				esc = strings.ReplaceAll(esc, "\n", "\\n")
				esc = strings.ReplaceAll(esc, "\r", "\\r")
				res += indent + "volt_buf_append(" + bufName + ", \"" + esc + "\");\n"
			} else {
				code, isTemp := c.transpileExpression(seg)
				if isTemp {
					res += indent + "{ VoltValue* _tmp = " + code + "; volt_buf_append_value(" + bufName + ", _tmp); volt_value_free(_tmp); }\n"
				} else {
					res += indent + "volt_buf_append_value(" + bufName + ", " + code + ");\n"
				}
			}
		}
		return res
	}
	code, isTemp := c.transpileExpression(expr)
	if isTemp {
		return indent + "{ VoltValue* _tmp = " + code + "; volt_buf_append_value(" + bufName + ", _tmp); volt_value_free(_tmp); }\n"
	}
	return indent + "volt_buf_append_value(" + bufName + ", " + code + ");\n"
}

func (c *Compiler) transpileExpression(expr ast.Expression) (string, bool) {
	if expr == nil {
		return "make_str(\"\")", true
	}
	switch e := expr.(type) {
	case *ast.InterpolatedStringLiteral:
		sb := "({ VoltBuffer* _b = volt_buf_new(); "
		sb += c.transpileToBuffer(e, "_b", "")
		sb += " VoltValue* _rv = make_str(_b->data); volt_buf_free(_b); _rv; })"
		return sb, true
	case *ast.Identifier:
		return e.Value, false
	case *ast.IntegerLiteral:
		return "make_int(" + strconv.FormatInt(e.Value, 10) + ")", true
	case *ast.StringLiteral:
		esc := strings.ReplaceAll(e.Value, "\\", "\\\\")
		esc = strings.ReplaceAll(esc, "\"", "\\\"")
		esc = strings.ReplaceAll(esc, "\n", "\\n")
		esc = strings.ReplaceAll(esc, "\r", "\\r")
		return "make_str(\"" + esc + "\")", true
	case *ast.Boolean:
		return "make_bool(" + strconv.FormatBool(e.Value) + ")", true
	case *ast.ArrayLiteral:
		elems := []string{}
		for _, el := range e.Elements {
			code, isTemp := c.transpileExpression(el)
			if isTemp {
				elems = append(elems, code)
			} else {
				elems = append(elems, "volt_value_copy("+code+")")
			}
		}
		inner := ""
		for i, el := range elems {
			inner += "a->a.elements[" + strconv.Itoa(i) + "] = " + el + "; "
		}
		return "({ VoltValue* a = malloc(sizeof(VoltValue)); a->type = TYPE_ARRAY; a->a.len = " + strconv.Itoa(len(elems)) + "; a->a.elements = malloc(" + strconv.Itoa(len(elems)) + " * sizeof(VoltValue*)); " + inner + " a; })", true
	case *ast.IndexExpression:
		itCode, itTemp := c.transpileExpression(e.Left)
		idxCode, isIdxTemp := c.transpileExpression(e.Index)

		res := "({ VoltValue* _it = " + itCode + "; VoltValue* _idx = " + idxCode + "; VoltValue* _rv = NULL; " +
			"if (_it->type == TYPE_ARRAY) { long long _i = _idx->i; _rv = (_i >= 0 && _i < _it->a.len) ? volt_value_copy(_it->a.elements[_i]) : make_str(\"\"); } " +
			"else { _rv = volt_value_copy(map_get(_it, to_str(_idx))); } " +
			(func() string {
				if isIdxTemp {
					return "volt_value_free(_idx); "
				}
				return ""
			}()) +
			(func() string {
				if itTemp {
					return "volt_value_free(_it); "
				}
				return ""
			}()) +
			"_rv; })"
		return res, true
	case *ast.InfixExpression:
		l, lTemp := c.transpileExpression(e.Left)
		r, rTemp := c.transpileExpression(e.Right)
		lArg := l
		if !lTemp {
			lArg = "volt_value_copy(" + l + ")"
		}
		rArg := r
		if !rTemp {
			rArg = "volt_value_copy(" + r + ")"
		}
		return "dynamic_add(" + lArg + ", " + rArg + ")", true
	case *ast.MethodCallExpression:
		objIdent, ok := e.Object.(*ast.Identifier)
		if ok {
			fullName := c.curPrefix + objIdent.Value + e.Method.Value
			if c.components[fullName] {
				args := []string{}
				for _, arg := range e.Arguments {
					code, isTemp := c.transpileExpression(arg)
					if isTemp {
						args = append(args, code)
					} else {
						args = append(args, "volt_value_copy("+code+")")
					}
				}
				argv := "NULL"
				if len(args) > 0 {
					argv = "({ VoltValue** v = malloc(" + strconv.Itoa(len(args)) + " * sizeof(VoltValue*)); "
					for i, arg := range args {
						argv += "v[" + strconv.Itoa(i) + "] = " + arg + "; "
					}
					argv += " v; })"
				}
				ctxParam := "NULL"
				if c.curComponent != "" {
					ctxParam = "ctx"
				} else {
					return "({ VoltBuffer* _b = volt_buf_new(); render_" + fullName + "(_b, " + strconv.Itoa(len(args)) + ", " + argv + "); if (_b->len > 0) printf(\"%s\\n\", _b->data); volt_buf_free(_b); make_str(\"\"); })", true
				}
				return "({ render_" + fullName + "(" + ctxParam + ", " + strconv.Itoa(len(args)) + ", " + argv + "); make_str(\"\"); })", true
			}
		}

		obj, objTemp := c.transpileExpression(e.Object)
		objArg := obj
		if !objTemp {
			objArg = "volt_value_copy(" + obj + ")"
		}
		switch e.Method.Value {
		case "trim":
			return "str_trim(" + objArg + ")", true
		case "upper":
			return "str_upper(" + objArg + ")", true
		}
	case *ast.CallExpression:
		var funcCode string
		var isFuncTemp bool
		var name string

		if nameIdent, ok := e.Function.(*ast.Identifier); ok {
			name = nameIdent.Value
		} else {
			funcCode, isFuncTemp = c.transpileExpression(e.Function)
		}

		argCodes := []string{}
		for _, arg := range e.Arguments {
			code, isTemp := c.transpileExpression(arg)
			if isTemp {
				argCodes = append(argCodes, code)
			} else {
				argCodes = append(argCodes, "volt_value_copy("+code+")")
			}
		}

		res := "({ "
		if isFuncTemp {
			res += "VoltValue* _f = " + funcCode + "; "
		}
		for i, code := range argCodes {
			res += fmt.Sprintf("VoltValue* _a%d = %s; ", i, code)
		}

		if len(argCodes) > 0 {
			res += fmt.Sprintf("VoltValue** _argv = malloc(%d * sizeof(VoltValue*)); ", len(argCodes))
			for i := range argCodes {
				res += fmt.Sprintf("_argv[%d] = _a%d; ", i, i)
			}
		} else {
			res += "VoltValue** _argv = NULL; "
		}

		compName := ""
		if name != "" {
			if c.components[c.curPrefix+name] {
				compName = c.curPrefix + name
			} else if alias, ok := c.ComponentAliases[name]; ok {
				compName = alias
			} else if c.components[name] {
				compName = name
			}
		}

		if compName != "" {
			ctxParam := "NULL"
			if c.curComponent != "" {
				ctxParam = "ctx"
			} else {
				res += "VoltBuffer* _b = volt_buf_new(); render_" + compName + "(_b, " + strconv.Itoa(len(argCodes)) + ", _argv); if (_b->len > 0) printf(\"%s\\n\", _b->data); volt_buf_free(_b); "
			}
			if c.curComponent != "" {
				res += "render_" + compName + "(" + ctxParam + ", " + strconv.Itoa(len(argCodes)) + ", _argv); "
			}
		} else if name != "" {
			switch name {
			case "print":
				res += "printf(\"%s\\n\", to_str(_a0)); "
			case "json_parse":
				res += "VoltValue* _rv = json_parse(to_str(_a0)); "
			case "db_save":
				res += "db_save(to_str(_a0), to_str(_a1)); "
			case "db_get":
				res += "VoltValue* _rv = db_get(to_str(_a0)); "
			case "file_write":
				res += "volt_file_write(to_str(_a0), to_str(_a1)); "
			case "get_addr":
				res += "char* _b = malloc(32); sprintf(_b, \"%p\", (void*)_a0); VoltValue* _rv = make_str(_b); free(_b); "
			case "exit":
				res += "exit((int)_a0->i); "
			case "sleep":
				res += "sleep((int)_a0->i); "
			default:
				if strings.HasPrefix(name, "render_") {
					cName := strings.TrimPrefix(name, "render_")
					ctxParam := "NULL"
					if c.curComponent != "" {
						ctxParam = "ctx"
					} else {
						res += "VoltBuffer* _b = volt_buf_new(); render_" + cName + "(_b, " + strconv.Itoa(len(argCodes)) + ", _argv); if (_b->len > 0) printf(\"%s\\n\", _b->data); volt_buf_free(_b); "
					}
					if c.curComponent != "" {
						res += "render_" + cName + "(" + ctxParam + ", " + strconv.Itoa(len(argCodes)) + ", _argv); "
					}
				} else {
					res += "VoltValue* _rv = " + name + "->f(" + strconv.Itoa(len(argCodes)) + ", _argv); "
				}
			}
		} else {
			res += "VoltValue* _rv = _f->f(" + strconv.Itoa(len(argCodes)) + ", _argv); "
		}

		// Cleanup
		if isFuncTemp {
			res += "volt_value_free(_f); "
		}
		for i := range argCodes {
			res += fmt.Sprintf("volt_value_free(_a%d); ", i)
		}
		if len(argCodes) > 0 {
			res += "free(_argv); "
		}

		if strings.Contains(res, "VoltValue* _rv =") {
			res += "_rv; })"
		} else {
			res += "make_str(\"\"); })"
		}
		return res, true
	}
	return "make_str(\"\")", true
}
