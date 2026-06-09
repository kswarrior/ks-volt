package compiler

import (
	"fmt"
	"ks-volt/ast"
	"strings"
)

type Compiler struct {
	globalVars []string
	funcID     int
}

func New() *Compiler {
	return &Compiler{}
}

func (c *Compiler) Compile(program *ast.Program) string {
	var sb strings.Builder

	// C Header and Runtime
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

#define MAX_TASKS 4096
#define NUM_WORKERS 4

typedef struct Task {
    void (*func)(void*);
    void* arg;
} Task;

typedef struct {
    Task queue[MAX_TASKS];
    int head;
    int tail;
    pthread_mutex_t lock;
} Processor;

Processor processors[NUM_WORKERS];
pthread_t workers[NUM_WORKERS];
__thread int worker_id;
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

char* dynamic_strcat(const char* s1, const char* s2) {
    char* res = malloc(strlen(s1) + strlen(s2) + 1);
    strcpy(res, s1);
    strcat(res, s2);
    return res;
}

void db_save(const char* key, const char* val) {
    pthread_mutex_lock(&db_file_lock);
    FILE* f = fopen("volt_db.json", "a+");
    if(f) {
        fprintf(f, "%s:%s\n", key, val);
        fclose(f);
    }
    pthread_mutex_unlock(&db_file_lock);
}

char* db_get(const char* key) {
    return "ready";
}

char* fetch_api(const char* url) {
    char cmd[1024];
    sprintf(cmd, "curl -s %s", url);
    FILE* fp = popen(cmd, "r");
    if (!fp) return "Error";
    char* res = malloc(4096);
    res[0] = '\0';
    char line[1024];
    while (fgets(line, sizeof(line), fp)) {
        strcat(res, line);
    }
    pclose(fp);
    return res;
}

void file_write(const char* filename, const char* data) {
    pthread_mutex_lock(&db_file_lock);
    FILE* f = fopen(filename, "w");
    if(f) {
        fputs(data, f);
        fclose(f);
    }
    pthread_mutex_unlock(&db_file_lock);
}

void* http_server(void* arg) {
    char* data = (char*)arg;
    int port = 8080;
    int server_fd = socket(AF_INET, SOCK_STREAM, 0);
    int opt = 1;
    setsockopt(server_fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));
    struct sockaddr_in address = {AF_INET, htons(port), INADDR_ANY};
    bind(server_fd, (struct sockaddr *)&address, sizeof(address));
    listen(server_fd, 10);
    while(1) {
        int client = accept(server_fd, NULL, NULL);
        char* resp = malloc(strlen(data) + 1024);
        sprintf(resp, "HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nContent-Length: %ld\r\n\r\n%s", strlen(data), data);
        send(client, resp, strlen(resp), 0);
        close(client);
        free(resp);
    }
    return NULL;
}

void serve_html(int port, char* html) {
    pthread_t t;
    pthread_create(&t, NULL, http_server, html);
}

void connect_bot(const char* ip, int port, void (*cb)(void*)) {
    int sock = socket(AF_INET, SOCK_STREAM, 0);
    struct sockaddr_in addr = {0};
    addr.sin_family = AF_INET;
    addr.sin_port = htons(port);
    inet_pton(AF_INET, ip, &addr.sin_addr);
    if (connect(sock, (struct sockaddr*)&addr, sizeof(addr)) == 0) {
        cb(NULL);
    } else {
        printf("🤖 Bot failed to connect to %s:%d\n", ip, port);
    }
    close(sock);
}

typedef struct {
    char* name;
    void (*handler)(void*);
} EventHandler;
EventHandler handlers[512];
int handlers_count = 0;
pthread_mutex_t event_lock = PTHREAD_MUTEX_INITIALIZER;

void on_event(char* name, void (*h)(void*)) {
    pthread_mutex_lock(&event_lock);
    if(handlers_count < 512) {
        handlers[handlers_count].name = name;
        handlers[handlers_count++].handler = h;
    }
    pthread_mutex_unlock(&event_lock);
}

void emit_event(char* name) {
    pthread_mutex_lock(&event_lock);
    for(int i=0; i<handlers_count; i++) {
        if(strcmp(handlers[i].name, name) == 0) {
            schedule_task(rand() % NUM_WORKERS, handlers[i].handler, NULL);
        }
    }
    pthread_mutex_unlock(&event_lock);
}

typedef struct {
    int ms;
    void (*func)(void*);
} IntervalArg;

void* interval_runner(void* arg) {
    IntervalArg* ia = (IntervalArg*)arg;
    while(1) {
        usleep(ia->ms * 1000);
        schedule_task(rand() % NUM_WORKERS, ia->func, NULL);
    }
    return NULL;
}

void start_interval(int ms, void (*func)(void*)) {
    IntervalArg* arg = malloc(sizeof(IntervalArg));
    arg->ms = ms;
    arg->func = func;
    pthread_t t;
    pthread_create(&t, NULL, interval_runner, arg);
}
`)

	// Scan for global variables first
	for _, stmt := range program.Statements {
		if as, ok := stmt.(*ast.AssignmentStatement); ok {
			c.globalVars = append(c.globalVars, as.Name.Value)
		}
	}

	for _, v := range c.globalVars {
		sb.WriteString(fmt.Sprintf("char* %s;\n", v))
	}

	// Generated Functions (Lambdas)
	var funcs strings.Builder

	sb.WriteString("\n// Program Logic\n")

	// Transpile main statements
	var mainBody strings.Builder
	for _, stmt := range program.Statements {
		mainBody.WriteString(c.transpileStatement(stmt, &funcs))
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
	sb.WriteString("    while(1) sleep(1);\n")
	sb.WriteString("    return 0;\n")
	sb.WriteString("}\n")

	return sb.String()
}

func (c *Compiler) transpileStatement(stmt ast.Statement, funcs *strings.Builder) string {
	switch s := stmt.(type) {
	case *ast.AssignmentStatement:
		val := c.transpileExpression(s.Value)
		return fmt.Sprintf("    %s = %s;\n", s.Name.Value, val)
	case *ast.ExpressionStatement:
		return fmt.Sprintf("    %s;\n", c.transpileExpression(s.Expression))
	case *ast.SpawnStatement:
		fName := fmt.Sprintf("volt_func_%d", c.funcID)
		c.funcID++
		funcs.WriteString(fmt.Sprintf("void %s(void* arg) {\n", fName))
		for _, bs := range s.Body.Statements {
			funcs.WriteString("    " + c.transpileStatement(bs, funcs))
		}
		funcs.WriteString("}\n")

		if s.Name.Value == "connect_bot" {
			ip := c.transpileExpression(s.Args[0])
			port := c.transpileExpression(s.Args[1])
			return fmt.Sprintf("    connect_bot(%s, atoi(%s), %s);\n", ip, port, fName)
		}
		if s.Name.Value == "on" {
			evt := c.transpileExpression(s.Args[0])
			return fmt.Sprintf("    on_event(%s, %s);\n", evt, fName)
		}
		if s.Name.Value == "interval" {
			ms := c.transpileExpression(s.Args[0])
			return fmt.Sprintf("    start_interval(atoi(%s), %s);\n", ms, fName)
		}

		return fmt.Sprintf("    schedule_task(rand() %% NUM_WORKERS, %s, NULL);\n", fName)
	}
	return ""
}

func (c *Compiler) transpileExpression(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.Identifier:
		return e.Value
	case *ast.IntegerLiteral:
		return fmt.Sprintf("\"%d\"", e.Value)
	case *ast.StringLiteral:
		return fmt.Sprintf("\"%s\"", e.Value)
	case *ast.InfixExpression:
		return fmt.Sprintf("dynamic_strcat(%s, %s)", c.transpileExpression(e.Left), c.transpileExpression(e.Right))
	case *ast.CallExpression:
		name := ""
		if ident, ok := e.Function.(*ast.Identifier); ok {
			name = ident.Value
		}
		args := []string{}
		for _, arg := range e.Arguments {
			args = append(args, c.transpileExpression(arg))
		}
		switch name {
		case "print":
			return fmt.Sprintf("printf(\"%%s\\n\", %s)", args[0])
		case "serve_html":
			return fmt.Sprintf("serve_html(atoi(%s), %s)", args[0], args[1])
		case "db_save":
			return fmt.Sprintf("db_save(%s, %s)", args[0], args[1])
		case "db_get":
			return fmt.Sprintf("db_get(%s)", args[0])
		case "fetch_api":
			return fmt.Sprintf("fetch_api(%s)", args[0])
		case "emit":
			return fmt.Sprintf("emit_event(%s)", args[0])
		case "file_write":
			return fmt.Sprintf("file_write(%s, %s)", args[0], args[1])
		}
	}
	return "\"\""
}
