package compiler

import (
	"ks-volt/ast"
	"strconv"
	"strings"
)

type Compiler struct {
	globalVars map[string]bool
	funcID     int
}

func New() *Compiler {
	return &Compiler{
		globalVars: make(map[string]bool),
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

#define MAX_TASKS 8192
#define NUM_WORKERS 4

typedef enum { TYPE_STR, TYPE_INT, TYPE_BOOL, TYPE_ARRAY, TYPE_MAP, TYPE_FN } VoltType;

struct VoltValue;
typedef struct VoltValue* (*VoltFn)(int, struct VoltValue**);

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
    v->type = TYPE_STR; v->s = strdup(s); return v;
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

char* to_str(VoltValue* v) {
    if (!v) return "";
    if (v->type == TYPE_STR) return v->s;
    char* buf = malloc(128);
    if (v->type == TYPE_INT) sprintf(buf, "%lld", v->i);
    else if (v->type == TYPE_BOOL) sprintf(buf, "%s", v->b ? "true" : "false");
    else if (v->type == TYPE_FN) sprintf(buf, "[function]");
    else return "complex";
    return buf;
}

VoltValue* dynamic_add(VoltValue* a, VoltValue* b) {
    char* s1 = to_str(a); char* s2 = to_str(b);
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

	sb.WriteString(funcs.String())
	sb.WriteString("\nint main() {\n")
	sb.WriteString("    srand(time(NULL));\n")
	sb.WriteString("    for(int i=0; i<NUM_WORKERS; i++) {\n")
	sb.WriteString("        int* id = malloc(sizeof(int)); *id = i;\n")
	sb.WriteString("        pthread_mutex_init(&processors[i].lock, NULL);\n")
	sb.WriteString("        pthread_create(&workers[i], NULL, worker_loop, id);\n")
	sb.WriteString("    }\n")
	sb.WriteString(mainBody.String())
	sb.WriteString("    while(1) sleep(1); return 0;\n}\n")

	return sb.String()
}

func (c *Compiler) collectGlobalVars(program *ast.Program) {
	var walker func(node interface{})
	walker = func(node interface{}) {
		if node == nil { return }
		switch n := node.(type) {
		case *ast.Program:
			for _, s := range n.Statements { walker(s) }
		case *ast.AssignmentStatement:
			c.globalVars[n.Name.Value] = true
			walker(n.Value)
		case *ast.FunctionStatement:
			c.globalVars[n.Name.Value] = true
			walker(n.Body)
		case *ast.BlockStatement:
			for _, s := range n.Statements { walker(s) }
		case *ast.LoopStatement:
			c.globalVars[n.Variable.Value] = true
			walker(n.Body)
		case *ast.TryCatchStatement:
			walker(n.TryBody)
			c.globalVars[n.CatchVariable.Value] = true
			walker(n.CatchBody)
		case *ast.IfStatement:
			walker(n.Consequence)
			if n.Alternative != nil { walker(n.Alternative) }
		case *ast.SpawnStatement:
			walker(n.Body)
		case *ast.InfixExpression:
			walker(n.Left); walker(n.Right)
		case *ast.CallExpression:
			walker(n.Function)
			for _, a := range n.Arguments { walker(a) }
		case *ast.IndexExpression:
			walker(n.Left); walker(n.Index)
		case *ast.ArrayLiteral:
			for _, e := range n.Elements { walker(e) }
		case *ast.ExpressionStatement:
			walker(n.Expression)
		case *ast.MethodCallExpression:
			walker(n.Object)
			for _, a := range n.Arguments { walker(a) }
		}
	}
	walker(program)
}

func (c *Compiler) transpileStatement(stmt ast.Statement, funcs *strings.Builder, indent string) string {
	if stmt == nil { return "" }
	switch s := stmt.(type) {
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
		for _, st := range s.Consequence.Statements { cons.WriteString(c.transpileStatement(st, funcs, indent+"    ")) }
		var alt strings.Builder
		if s.Alternative != nil {
			for _, st := range s.Alternative.Statements { alt.WriteString(c.transpileStatement(st, funcs, indent+"    ")) }
		}
		res := indent + "if (" + cond + "->b) {\n" + cons.String() + indent + "}"
		if s.Alternative != nil {
			res += " else {\n" + alt.String() + indent + "}"
		}
		return res + "\n"
	case *ast.ExpressionStatement:
		return indent + c.transpileExpression(s.Expression) + ";\n"
	case *ast.LoopStatement:
		it := c.transpileExpression(s.Iterable)
		var body strings.Builder
		for _, bs := range s.Body.Statements { body.WriteString(c.transpileStatement(bs, funcs, indent+"    ")) }
		return indent + "for(int i=0; i<" + it + "->a.len; i++) {\n" +
			indent + "    " + s.Variable.Value + " = " + it + "->a.elements[i];\n" +
			body.String() + indent + "}\n"
	case *ast.TryCatchStatement:
		id := strconv.Itoa(c.funcID); c.funcID++
		funcs.WriteString("void volt_try_" + id + "(void* arg) {\n")
		for _, ts := range s.TryBody.Statements { funcs.WriteString(c.transpileStatement(ts, funcs, "    ")) }
		funcs.WriteString("}\n")
		return indent + "{ jmp_buf env_" + id + "; current_jmp_env = &env_" + id + ";\n" +
			indent + "if (setjmp(env_" + id + ") == 0) {\n" +
			indent + "    volt_try_" + id + "(NULL);\n" +
			indent + "} else {\n" +
			indent + "    " + s.CatchVariable.Value + " = make_str(\"OS Exception\");\n" +
			c.transpileStatement(s.CatchBody, funcs, indent+"    ") + indent + "} }\n"
	case *ast.SpawnStatement:
		id := strconv.Itoa(c.funcID); c.funcID++
		funcs.WriteString("void volt_func_" + id + "(void* arg) {\n")
		for _, bs := range s.Body.Statements { funcs.WriteString(c.transpileStatement(bs, funcs, "    ")) }
		funcs.WriteString("}\n")
		if s.Name.Value == "connect_bot" {
			return indent + "connect_bot(to_str(" + c.transpileExpression(s.Args[0]) + "), (int)" + c.transpileExpression(s.Args[1]) + "->i, volt_func_" + id + ");\n"
		}
		if s.Name.Value == "interval" {
			return indent + "start_interval((int)" + c.transpileExpression(s.Args[0]) + "->i, volt_func_" + id + ");\n"
		}
		return indent + "schedule_task(rand() % NUM_WORKERS, volt_func_" + id + ", NULL);\n"
	}
	return ""
}

func (c *Compiler) transpileExpression(expr ast.Expression) string {
	if expr == nil { return "make_str(\"\")" }
	switch e := expr.(type) {
	case *ast.Identifier: return e.Value
	case *ast.IntegerLiteral: return "make_int(" + strconv.FormatInt(e.Value, 10) + ")"
	case *ast.StringLiteral:
		esc := strings.ReplaceAll(e.Value, "\\", "\\\\")
		esc = strings.ReplaceAll(esc, "\"", "\\\"")
		return "make_str(\"" + esc + "\")"
	case *ast.Boolean: return "make_bool(" + strconv.FormatBool(e.Value) + ")"
	case *ast.ArrayLiteral:
		elems := []string{}
		for _, el := range e.Elements { elems = append(elems, c.transpileExpression(el)) }
		inner := ""
		for i, el := range elems { inner += "a->a.elements[" + strconv.Itoa(i) + "] = " + el + "; " }
		return "({ VoltValue* a = malloc(sizeof(VoltValue)); a->type = TYPE_ARRAY; a->a.len = " + strconv.Itoa(len(elems)) + "; a->a.elements = malloc(" + strconv.Itoa(len(elems)) + " * sizeof(VoltValue*)); " + inner + " a; })"
	case *ast.IndexExpression:
		it := c.transpileExpression(e.Left)
		idx := c.transpileExpression(e.Index)
		return "((" + it + "->type == TYPE_ARRAY) ? " + it + "->a.elements[" + idx + "->i] : map_get(" + it + ", to_str(" + idx + ")))"
	case *ast.InfixExpression: return "dynamic_add(" + c.transpileExpression(e.Left) + ", " + c.transpileExpression(e.Right) + ")"
	case *ast.MethodCallExpression:
		obj := c.transpileExpression(e.Object)
		switch e.Method.Value {
		case "trim": return "str_trim(" + obj + ")"
		case "upper": return "str_upper(" + obj + ")"
		}
	case *ast.CallExpression:
		nameIdent, ok := e.Function.(*ast.Identifier)
		if !ok {
			fexpr := c.transpileExpression(e.Function)
			args := []string{}
			for _, arg := range e.Arguments { args = append(args, c.transpileExpression(arg)) }
			argv := "({ VoltValue** v = malloc(" + strconv.Itoa(len(args)) + " * sizeof(VoltValue*)); "
			for i, arg := range args { argv += "v[" + strconv.Itoa(i) + "] = " + arg + "; " }
			argv += " v; })"
			return fexpr + "->f(" + strconv.Itoa(len(args)) + ", " + argv + ")"
		}
		name := nameIdent.Value
		args := []string{}
		for _, arg := range e.Arguments { args = append(args, c.transpileExpression(arg)) }
		switch name {
		case "print": return "printf(\"%s\\n\", to_str(" + args[0] + "))"
		case "json_parse": return "json_parse(to_str(" + args[0] + "))"
		case "db_save": return "db_save(to_str(" + args[0] + "), to_str(" + args[1] + "))"
		case "db_get": return "db_get(to_str(" + args[0] + "))"
		case "file_write": return "volt_file_write(to_str(" + args[0] + "), to_str(" + args[1] + "))"
		case "get_addr": return "make_str(({ char* b = malloc(32); sprintf(b, \"%p\", (void*)" + args[0] + "); b; }))"
		default:
			argv := "NULL"
			if len(args) > 0 {
				argv = "({ VoltValue** v = malloc(" + strconv.Itoa(len(args)) + " * sizeof(VoltValue*)); "
				for i, arg := range args { argv += "v[" + strconv.Itoa(i) + "] = " + arg + "; " }
				argv += " v; })"
			}
			return name + "->f(" + strconv.Itoa(len(args)) + ", " + argv + ")"
		}
	}
	return "make_str(\"\")"
}
