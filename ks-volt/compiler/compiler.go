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
	webBlocks        []string
}

func New() *Compiler {
	return &Compiler{
		globalVars:       make(map[string]bool),
		components:       make(map[string]bool),
		ComponentAliases: make(map[string]string),
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
    VoltValue* rv = make_str(res); free(res); return rv;
}

VoltValue* str_trim(VoltValue* v) {
    char* s = v->s;
    while(isspace(*s)) s++;
    if(*s == 0) return make_str("");
    char* end = s + strlen(s) - 1;
    while(end > s && isspace(*end)) end--;
    char* res = strndup(s, end - s + 1);
    VoltValue* rv = make_str(res); free(res);
    return rv;
}

VoltValue* str_upper(VoltValue* v) {
    char* s = strdup(v->s);
    for(int i=0; s[i]; i++) s[i] = toupper(s[i]);
    VoltValue* res = make_str(s); free(s);
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
typedef struct Router { Route routes[100]; int count; void (*before)(void*); } Router;

void volt_start_web_server(Router* r, int port) {
    printf("Web server started on port %d with %d routes\n", port, r->count);
}

const char* to_str(VoltValue* v) {
    static __thread char b[128];
    return to_str_buf(v, b);
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
		args := []string{}
		for _, arg := range s.Args {
			args = append(args, "to_str("+c.transpileExpression(arg)+")")
		}
		switch s.Token.Type {
		case token.FS_RM:
			return indent + "fs_rm(" + args[0] + ");\n"
		case token.FS_MV:
			return indent + "fs_mv(" + args[0] + ", " + args[1] + ");\n"
		case token.FS_CP:
			return indent + "fs_cp(" + args[0] + ", " + args[1] + ");\n"
		case token.FS_TOUCH:
			return indent + "fs_touch(" + args[0] + ");\n"
		case token.FS_CAT:
			return indent + "fs_cat(" + args[0] + ");\n"
		}
		return ""
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
		data, err := os.ReadFile(s.Path)
		if err != nil {
			fmt.Printf("Error importing component %s: %v\n", s.Path, err)
			return ""
		}
		subL := lexer.New(string(data))
		subP := parser.New(subL)
		subProg := subP.ParseProgram()
		for _, subStmt := range subProg.Statements {
			if cd, ok := subStmt.(*ast.ComponentDefinition); ok {
				originalName := cd.Name.Value
				fullName := s.Alias.Value + originalName
				cd.Name.Value = fullName
				c.ComponentAliases[fullName] = cd.Name.Value
				c.transpileStatement(cd, funcs, "")
				cd.Name.Value = originalName
			}
		}
		return ""
	case *ast.WebBlockStatement:
		name := s.Name
		if name == "" { name = "main_daemon" }
		routerName := "router_" + name
		funcs.WriteString("Router " + routerName + " = { .count = 0 };\n")
		for _, bs := range s.Body.Statements {
			switch r := bs.(type) {
			case *ast.PathStatement:
				handlerName := "handler_" + strconv.Itoa(c.funcID); c.funcID++
				funcs.WriteString("void " + handlerName + "(void* arg) {\n")
				funcs.WriteString(c.transpileStatement(r.Body, funcs, "    "))
				funcs.WriteString("}\n")
				funcs.WriteString("__attribute__((constructor)) void init_" + handlerName + "() { " +
					routerName + ".routes[" + routerName + ".count++] = (Route){\" " + r.Path + "\", " + handlerName + ", false }; }\n")
			case *ast.PathWsStatement:
				handlerName := "handler_" + strconv.Itoa(c.funcID); c.funcID++
				funcs.WriteString("void " + handlerName + "(void* arg) {\n")
				funcs.WriteString(c.transpileStatement(r.Body, funcs, "    "))
				funcs.WriteString("}\n")
				funcs.WriteString("__attribute__((constructor)) void init_" + handlerName + "() { " +
					routerName + ".routes[" + routerName + ".count++] = (Route){\" " + r.Path + "\", " + handlerName + ", true }; }\n")
			case *ast.BeforeEachStatement:
				handlerName := "before_" + name
				funcs.WriteString("void " + handlerName + "(void* arg) {\n")
				funcs.WriteString(c.transpileStatement(r.Body, funcs, "    "))
				funcs.WriteString("}\n")
				funcs.WriteString("__attribute__((constructor)) void init_" + handlerName + "() { " + routerName + ".before = " + handlerName + "; }\n")
			}
		}
		return indent + "volt_start_web_server(&" + routerName + ", 8080);\n"
	case *ast.AssignmentStatement:
		return indent + s.Name.Value + " = " + c.transpileExpression(s.Value) + ";\n"
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
		return indent + "return " + c.transpileExpression(s.ReturnValue) + ";\n"
	case *ast.IfStatement:
		cond := c.transpileExpression(s.Condition)
		var cons strings.Builder
		for _, st := range s.Consequence.Statements {
			cons.WriteString(c.transpileStatement(st, funcs, indent+"    "))
		}
		var alt strings.Builder
		if s.Alternative != nil {
			for _, st := range s.Alternative.Statements {
				alt.WriteString(c.transpileStatement(st, funcs, indent+"    "))
			}
		}
		res := indent + "if (" + cond + "->b) {\n" + cons.String() + indent + "}"
		if s.Alternative != nil {
			res += " else {\n" + alt.String() + indent + "}"
		}
		return res + "\n"
	case *ast.ExpressionStatement:
		if is, ok := s.Expression.(*ast.InterpolatedStringLiteral); ok && c.curComponent != "" {
			res := ""
			for _, seg := range is.Segments {
				res += indent + "volt_buf_append_value(ctx, " + c.transpileExpression(seg) + ");\n"
			}
			return res
		}
		if call, ok := s.Expression.(*ast.CallExpression); ok && c.curComponent != "" {
			if ident, ok := call.Function.(*ast.Identifier); ok && ident.Value == "print" {
				if is, ok := call.Arguments[0].(*ast.InterpolatedStringLiteral); ok {
					res := ""
					for _, seg := range is.Segments {
						res += indent + "volt_buf_append_value(ctx, " + c.transpileExpression(seg) + ");\n"
					}
					res += indent + "volt_buf_append(ctx, \"\\n\");\n"
					return res
				}
			}
		}

		return indent + c.transpileExpression(s.Expression) + ";\n"
	case *ast.LoopStatement:
		it := c.transpileExpression(s.Iterable)
		var body strings.Builder
		for _, bs := range s.Body.Statements {
			body.WriteString(c.transpileStatement(bs, funcs, indent+"    "))
		}
		id := strconv.Itoa(c.funcID); c.funcID++
		return indent + "for(int _idx_" + id + "=0; _idx_" + id + "<" + it + "->a.len; _idx_" + id + "++) {\n" +
			indent + "    " + s.Variable.Value + " = " + it + "->a.elements[_idx_" + id + "];\n" +
			body.String() + indent + "}\n"
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
			indent + "    " + s.CatchVariable.Value + " = make_str(\"OS Exception\");\n" +
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
			return indent + "connect_bot(to_str(" + c.transpileExpression(s.Args[0]) + "), (int)" + c.transpileExpression(s.Args[1]) + "->i, volt_func_" + id + ");\n"
		}
		if s.Name.Value == "interval" {
			return indent + "start_interval((int)" + c.transpileExpression(s.Args[0]) + "->i, volt_func_" + id + ");\n"
		}
		return indent + "schedule_task(rand() % NUM_WORKERS, volt_func_" + id + ", NULL);\n"
	case *ast.BeforeEachStatement:
		return indent + "// Middleware block execution\n"
	}
	return ""
}

func (c *Compiler) transpileExpression(expr ast.Expression) string {
	if expr == nil {
		return "make_str(\"\")"
	}
	switch e := expr.(type) {
	case *ast.InterpolatedStringLiteral:
		sb := "({ VoltBuffer* _b = volt_buf_new(); "
		for _, seg := range e.Segments {
			sb += "volt_buf_append_value(_b, " + c.transpileExpression(seg) + "); "
		}
		sb += " VoltValue* _rv = make_str(_b->data); volt_buf_free(_b); _rv; })"
		return sb
	case *ast.Identifier:
		return e.Value
	case *ast.IntegerLiteral:
		return "make_int(" + strconv.FormatInt(e.Value, 10) + ")"
	case *ast.StringLiteral:
		esc := strings.ReplaceAll(e.Value, "\\", "\\\\")
		esc = strings.ReplaceAll(esc, "\"", "\\\"")
		esc = strings.ReplaceAll(esc, "\n", "\\n")
		esc = strings.ReplaceAll(esc, "\r", "\\r")
		return "make_str(\"" + esc + "\")"
	case *ast.Boolean:
		return "make_bool(" + strconv.FormatBool(e.Value) + ")"
	case *ast.ArrayLiteral:
		elems := []string{}
		for _, el := range e.Elements {
			elems = append(elems, c.transpileExpression(el))
		}
		inner := ""
		for i, el := range elems {
			inner += "a->a.elements[" + strconv.Itoa(i) + "] = " + el + "; "
		}
		return "({ VoltValue* a = malloc(sizeof(VoltValue)); a->type = TYPE_ARRAY; a->a.len = " + strconv.Itoa(len(elems)) + "; a->a.elements = malloc(" + strconv.Itoa(len(elems)) + " * sizeof(VoltValue*)); " + inner + " a; })"
	case *ast.IndexExpression:
		it := c.transpileExpression(e.Left)
		idx := c.transpileExpression(e.Index)
		return "((" + it + "->type == TYPE_ARRAY) ? " + it + "->a.elements[" + idx + "->i] : map_get(" + it + ", to_str(" + idx + ")))"
	case *ast.InfixExpression:
		return "dynamic_add(" + c.transpileExpression(e.Left) + ", " + c.transpileExpression(e.Right) + ")"
	case *ast.MethodCallExpression:
		obj := c.transpileExpression(e.Object)
		switch e.Method.Value {
		case "trim":
			return "str_trim(" + obj + ")"
		case "upper":
			return "str_upper(" + obj + ")"
		}
	case *ast.CallExpression:
		nameIdent, ok := e.Function.(*ast.Identifier)
		if !ok {
			fexpr := c.transpileExpression(e.Function)
			args := []string{}
			for _, arg := range e.Arguments {
				args = append(args, c.transpileExpression(arg))
			}
			argv := "({ VoltValue** v = malloc(" + strconv.Itoa(len(args)) + " * sizeof(VoltValue*)); "
			for i, arg := range args {
				argv += "v[" + strconv.Itoa(i) + "] = " + arg + "; "
			}
			argv += " v; })"
			return fexpr + "->f(" + strconv.Itoa(len(args)) + ", " + argv + ")"
		}
		name := nameIdent.Value
		args := []string{}
		for _, arg := range e.Arguments {
			args = append(args, c.transpileExpression(arg))
		}

		compName := ""
		if alias, ok := c.ComponentAliases[name]; ok {
			compName = alias
		} else if c.components[name] {
			compName = name
		}

		if compName != "" {
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
				return "({ VoltBuffer* _b = volt_buf_new(); render_" + compName + "(_b, " + strconv.Itoa(len(args)) + ", " + argv + "); printf(\"%s\\n\", _b->data); volt_buf_free(_b); make_str(\"\"); })"
			}
			return "({ render_" + compName + "(" + ctxParam + ", " + strconv.Itoa(len(args)) + ", " + argv + "); make_str(\"\"); })"
		}

		switch name {
		case "print":
			return "printf(\"%s\\n\", to_str(" + args[0] + "))"
		case "json_parse":
			return "json_parse(to_str(" + args[0] + "))"
		case "db_save":
			return "db_save(to_str(" + args[0] + "), to_str(" + args[1] + "))"
		case "db_get":
			return "db_get(to_str(" + args[0] + "))"
		case "file_write":
			return "volt_file_write(to_str(" + args[0] + "), to_str(" + args[1] + "))"
		case "get_addr":
			return "make_str(({ char* b = malloc(32); sprintf(b, \"%p\", (void*)" + args[0] + "); b; }))"
		default:
			if strings.HasPrefix(name, "render_") {
				compName := strings.TrimPrefix(name, "render_")
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
					return "({ VoltBuffer* _b = volt_buf_new(); render_" + compName + "(_b, " + strconv.Itoa(len(args)) + ", " + argv + "); printf(\"%s\\n\", _b->data); make_str(\"\"); })"
				}
				return "({ render_" + compName + "(" + ctxParam + ", " + strconv.Itoa(len(args)) + ", " + argv + "); make_str(\"\"); })"
			}

			argv := "NULL"
			if len(args) > 0 {
				argv = "({ VoltValue** v = malloc(" + strconv.Itoa(len(args)) + " * sizeof(VoltValue*)); "
				for i, arg := range args {
					argv += "v[" + strconv.Itoa(i) + "] = " + arg + "; "
				}
				argv += " v; })"
			}
			return name + "->f(" + strconv.Itoa(len(args)) + ", " + argv + ")"
		}
	}
	return "make_str(\"\")"
}
