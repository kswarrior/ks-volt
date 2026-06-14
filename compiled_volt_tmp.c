#include <stdio.h>
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
#include <stdatomic.h>

#define MAX_TASKS 8192
#define NUM_WORKERS 4

typedef enum { TYPE_STR, TYPE_INT, TYPE_BOOL, TYPE_ARRAY, TYPE_MAP, TYPE_FN, TYPE_RESULT } VoltType;

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

#define VOLT_BUF_CHECK(b, l) if ((b)->len + (l) >= (b)->cap) volt_buf_grow((b), (l))

void volt_buf_grow(VoltBuffer* b, size_t needed) {
    b->cap = (b->len + needed + 4096) * 2;
    b->data = realloc(b->data, b->cap);
}

void volt_buf_append(VoltBuffer* b, const char* s) {
    if (!b || !s) return;
    size_t slen = strlen(s);
    VOLT_BUF_CHECK(b, slen);
    memcpy(b->data + b->len, s, slen);
    b->len += slen;
    b->data[b->len] = '\0';
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
        struct { bool is_ok; struct VoltValue* val; } res;
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

VoltValue* make_ok(VoltValue* v) {
    VoltValue* rv = malloc(sizeof(VoltValue));
    rv->type = TYPE_RESULT; rv->res.is_ok = true; rv->res.val = v; return rv;
}

VoltValue* make_err(VoltValue* v) {
    VoltValue* rv = malloc(sizeof(VoltValue));
    rv->type = TYPE_RESULT; rv->res.is_ok = false; rv->res.val = v; return rv;
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
        case TYPE_RESULT:
            res->res.is_ok = v->res.is_ok;
            res->res.val = volt_value_copy(v->res.val);
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
    } else if (v->type == TYPE_RESULT) {
        volt_value_free(v->res.val);
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

// Thread-Safe Atomic Ring Buffer
typedef struct {
    atomic_int head;
    atomic_int tail;
    void* buffer[MAX_TASKS];
} VoltRingBuffer;

void volt_rb_push(VoltRingBuffer* rb, void* item) {
    int next = (atomic_load(&rb->tail) + 1) % MAX_TASKS;
    while (next == atomic_load(&rb->head));
    rb->buffer[atomic_load(&rb->tail)] = item;
    atomic_store(&rb->tail, next);
}

void* volt_rb_pop(VoltRingBuffer* rb) {
    if (atomic_load(&rb->head) == atomic_load(&rb->tail)) return NULL;
    void* item = rb->buffer[atomic_load(&rb->head)];
    atomic_store(&rb->head, (atomic_load(&rb->head) + 1) % MAX_TASKS);
    return item;
}

// Routing Logic
typedef struct {
    int client_fd;
    char* method;
    char* path;
    char* headers[50];
    int header_count;
    int status;
    bool is_fragment;
    VoltBuffer* response_body;
} VoltContext;

typedef struct Route { char* path; void (*handler)(VoltContext*); bool is_ws; } Route;
typedef struct Router { char* name; Route routes[100]; int count; void (*before)(VoltContext*); } Router;

__thread VoltContext* current_web_ctx = NULL;

VoltValue* volt_request_header(const char* name) {
    if (!current_web_ctx) return make_str("");
    for(int i=0; i<current_web_ctx->header_count; i++) {
        if (strncasecmp(current_web_ctx->headers[i], name, strlen(name)) == 0) {
            char* val = strchr(current_web_ctx->headers[i], ':');
            if (val) return make_str(val + 2);
        }
    }
    return make_str("");
}

void volt_redirect(const char* url) {
    if (!current_web_ctx) return;
    current_web_ctx->status = 302;
    char buf[512];
    sprintf(buf, "Location: %s\r\n", url);
    current_web_ctx->headers[current_web_ctx->header_count++] = strdup(buf);
}

VoltValue* volt_json(VoltValue* v) {
    if (!current_web_ctx) return v;
    current_web_ctx->headers[current_web_ctx->header_count++] = strdup("Content-Type: application/json\r\n");
    return v;
}

typedef struct { Router* r; int client_fd; } ConnTaskArgs;

void volt_dispatch_route(void* arg) {
    ConnTaskArgs* args = (ConnTaskArgs*)arg;
    VoltContext* ctx = malloc(sizeof(VoltContext));
    ctx->client_fd = args->client_fd;
    ctx->status = 200;
    ctx->is_fragment = false;
    ctx->header_count = 0;
    ctx->response_body = volt_buf_new();
    current_web_ctx = ctx;

    char buffer[2048];
    int n = read(ctx->client_fd, buffer, 2047);
    if (n > 0) {
        buffer[n] = '\0';
        char* method_str = strtok(buffer, " ");
        char* path_str = strtok(NULL, " ");
        ctx->method = strdup(method_str ? method_str : "GET");
        ctx->path = strdup(path_str ? path_str : "/");

        Route* target = NULL;
        for(int i=0; i<args->r->count; i++) {
            if (strcmp(args->r->routes[i].path, ctx->path) == 0) {
                target = &args->r->routes[i];
                break;
            }
        }

        if (args->r->before) args->r->before(ctx);

        if (target) {
            if (target->is_ws) {
                write(ctx->client_fd, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n", 78);
                target->handler(ctx);
            } else {
                target->handler(ctx);
                if (ctx->status == 302) {
                    char header[512];
                    sprintf(header, "HTTP/1.1 302 Found\r\nContent-Length: 0\r\n");
                    write(ctx->client_fd, header, strlen(header));
                    for(int i=0; i<ctx->header_count; i++) write(ctx->client_fd, ctx->headers[i], strlen(ctx->headers[i]));
                    write(ctx->client_fd, "\r\n", 2);
                } else if (ctx->is_fragment) {
                    write(ctx->client_fd, ctx->response_body->data, ctx->response_body->len);
                } else {
                    char header[512];
                    sprintf(header, "HTTP/1.1 %d OK\r\nContent-Length: %zu\r\n", ctx->status, ctx->response_body->len);
                    write(ctx->client_fd, header, strlen(header));
                    for(int i=0; i<ctx->header_count; i++) write(ctx->client_fd, ctx->headers[i], strlen(ctx->headers[i]));
                    write(ctx->client_fd, "\r\n", 2);
                    write(ctx->client_fd, ctx->response_body->data, ctx->response_body->len);
                }
            }
        } else {
            write(ctx->client_fd, "HTTP/1.1 404 Not Found\r\nContent-Length: 9\r\n\r\nNot Found", 53);
        }
        free(ctx->method); free(ctx->path);
    }
cleanup:
    close(ctx->client_fd);
    volt_buf_free(ctx->response_body);
    free(ctx);
    free(args);
}

void* volt_accept_loop(void* arg) {
    Router* r = (Router*)arg;
    int server_fd = socket(AF_INET, SOCK_STREAM, 0);
    struct sockaddr_in addr = { .sin_family = AF_INET, .sin_addr.s_addr = INADDR_ANY, .sin_port = htons(8080) };
    int opt = 1; setsockopt(server_fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));
    bind(server_fd, (struct sockaddr*)&addr, sizeof(addr));
    listen(server_fd, 100);
    while(1) {
        int client = accept(server_fd, NULL, NULL);
        ConnTaskArgs* args = malloc(sizeof(ConnTaskArgs));
        args->r = r; args->client_fd = client;
        schedule_task(rand() % NUM_WORKERS, volt_dispatch_route, args);
    }
    return NULL;
}

void volt_start_web_server(Router* r, int port) {
    pthread_t t;
    pthread_create(&t, NULL, volt_accept_loop, r);
    printf("KS-Panel Engine: Web server '%s' started on port %d\n", r->name, port);
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
#include "deps/quickjs.h"
#include "deps/quickjs-libc.h"
void render_UILoginPage(VoltBuffer* ctx, int argc, VoltValue** argv) {
    volt_buf_append(ctx, "<html>\n        <head><title>Login - KS-Volt Panel</title></head>\n        <body>\n            <h1>Login</h1>\n            <form action=\"/login\" method=\"POST\">\n                <input type=\"text\" name=\"username\" placeholder=\"Username\" /><br>\n                <input type=\"password\" name=\"password\" placeholder=\"Password\" /><br>\n                <button type=\"submit\">Login</button>\n            </form>\n        </body>\n    </html>");
}
void render_UIDashboard(VoltBuffer* ctx, int argc, VoltValue** argv) {
    VoltValue* user = argv[0];
    volt_buf_append(ctx, "<html>\n        <head><title>Dashboard - KS-Volt Panel</title></head>\n        <body>\n            <h1>Welcome, ");
    volt_buf_append_value(ctx, user);
    volt_buf_append(ctx, "!</h1>\n            <p>This is your KS-Volt powered dashboard.</p>\n            <nav>\n                <a href=\"/logout\">Logout</a>\n            </nav>\n        </body>\n    </html>");
}
Router router_panel_server = { .name = "panel_server", .count = 0 };
void before_panel_server_0(VoltContext* ctx) {
    volt_value_free(({ VoltValue* _rv = NULL; VoltValue* _a0 = make_str("Incoming request to panel..."); VoltValue** _argv = malloc(1 * sizeof(VoltValue*)); _argv[0] = _a0; printf("%s\n", to_str(_a0)); volt_value_free(_a0); free(_argv); _rv; })); if (current_web_ctx && current_web_ctx->status == 302) return;
}
__attribute__((constructor)) void init_before_panel_server_0() { router_panel_server.before = before_panel_server_0; }
void handler_1(VoltContext* ctx) {
    volt_value_free(({ render_UILoginPage(ctx->response_body, 0, NULL); make_str(""); })); if (current_web_ctx && current_web_ctx->status == 302) return;
}
__attribute__((constructor)) void init_handler_1() { router_panel_server.routes[router_panel_server.count++] = (Route){"/", handler_1, false }; }
void handler_2(VoltContext* ctx) {
    volt_value_free(({ render_UIDashboard(ctx->response_body, 1, ({ VoltValue** v = malloc(1 * sizeof(VoltValue*)); v[0] = make_str("Admin");  v; })); make_str(""); })); if (current_web_ctx && current_web_ctx->status == 302) return;
}
__attribute__((constructor)) void init_handler_2() { router_panel_server.routes[router_panel_server.count++] = (Route){"/dashboard", handler_2, false }; }
void handler_3(VoltContext* ctx) {
    volt_value_free(({ VoltValue* _rv = NULL; VoltValue* _a0 = make_str("Login attempt detected."); VoltValue** _argv = malloc(1 * sizeof(VoltValue*)); _argv[0] = _a0; printf("%s\n", to_str(_a0)); volt_value_free(_a0); free(_argv); _rv; })); if (current_web_ctx && current_web_ctx->status == 302) return;
    volt_value_free(({ VoltValue* _rv = NULL; VoltValue* _a0 = make_str("/dashboard"); VoltValue** _argv = malloc(1 * sizeof(VoltValue*)); _argv[0] = _a0; volt_redirect(to_str(_a0)); _rv = make_str(""); volt_value_free(_a0); free(_argv); _rv; })); if (current_web_ctx && current_web_ctx->status == 302) return;
}
__attribute__((constructor)) void init_handler_3() { router_panel_server.routes[router_panel_server.count++] = (Route){"/login", handler_3, false }; }
void handler_4(VoltContext* ctx) {
    ctx->is_fragment = true;
    ctx->response_body->len = 0; ctx->response_body->data[0] = '\0';
    volt_buf_append(ctx->response_body, "<div>Only this fragment</div>");
}
__attribute__((constructor)) void init_handler_4() { router_panel_server.routes[router_panel_server.count++] = (Route){"/fragment", handler_4, false }; }

int main(int argc, char** argv) {
    srand(time(NULL));
    for(int i=0; i<NUM_WORKERS; i++) {
        int* id = malloc(sizeof(int)); *id = i;
        pthread_mutex_init(&processors[i].lock, NULL);
        pthread_create(&workers[i], NULL, worker_loop, id);
    }
    volt_start_web_server(&router_panel_server, 8080);
    volt_value_free(({ VoltValue* _rv = NULL; VoltValue* _a0 = make_str("Panel server initialized on port 8080"); VoltValue** _argv = malloc(1 * sizeof(VoltValue*)); _argv[0] = _a0; printf("%s\n", to_str(_a0)); volt_value_free(_a0); free(_argv); _rv; })); if (current_web_ctx && current_web_ctx->status == 302) return;
    while(1) sleep(1); return 0;
}
