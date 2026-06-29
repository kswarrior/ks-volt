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
#include <stddef.h>

#ifndef _WIN32
#include <ucontext.h>
#include <sys/epoll.h>
#include <dlfcn.h>
#else
#include <windows.h>
#endif

struct VoltGreenThread;
extern __thread struct VoltGreenThread* current_green_thread;

#define MAX_TASKS 8192
#define NUM_WORKERS 4
#define STACK_SIZE 65536

typedef enum { TYPE_STR, TYPE_INT, TYPE_BOOL, TYPE_ARRAY, TYPE_MAP, TYPE_FN, TYPE_RESULT } VoltType;

struct VoltValue;
typedef struct VoltValue* (*VoltFn)(int, struct VoltValue**);

typedef struct VoltBuffer {
    char* data;
    size_t len;
    size_t cap;
    size_t high_watermark;
    bool paused;
} VoltBuffer;

VoltBuffer* volt_buf_new() {
    VoltBuffer* b = malloc(sizeof(VoltBuffer));
    b->cap = 4096; b->len = 0; b->data = malloc(b->cap);
    b->data[0] = '\0';
    b->high_watermark = 1024 * 1024;
    b->paused = false;
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

    if (b->len > b->high_watermark && current_green_thread && current_green_thread->fd != -1) {
        b->paused = true;
        struct epoll_event ev = { .events = EPOLLOUT | EPOLLONESHOT, .data.ptr = current_green_thread };
        epoll_ctl(global_epoll_fd, EPOLL_CTL_MOD, current_green_thread->fd, &ev);
        current_green_thread->state = THREAD_PARKED;
        volt_scheduler_yield();
    }
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

typedef enum { THREAD_READY, THREAD_RUNNING, THREAD_PARKED, THREAD_FINISHED } ThreadState;

#define MAX_FRAMES 128
typedef struct KVFrame {
    const char* func_name;
    const char* file;
    int line;
    jmp_buf env;
} KVFrame;

typedef void (*VoltYieldFn)(VoltBuffer*);

typedef struct VoltGreenThread {
    ucontext_t context;
    void* stack;
    void (*func)(void*);
    void* arg;
    int fd;
    ThreadState state;
    KVFrame frames[MAX_FRAMES];
    int frame_ptr;
    VoltYieldFn current_yield;
} VoltGreenThread;

extern __thread VoltGreenThread* current_green_thread;

typedef struct {
    VoltGreenThread* queue[MAX_TASKS];
    int head, tail;
    pthread_mutex_t lock;
    ucontext_t worker_context;
} Processor;

Processor processors[NUM_WORKERS];
pthread_t workers[NUM_WORKERS];
__thread int worker_id;
__thread VoltGreenThread* current_green_thread = NULL;

struct Router;

void kv_push_frame(const char* fn, const char* file, int line) {
    if (current_green_thread && current_green_thread->frame_ptr < MAX_FRAMES - 1) {
        current_green_thread->frame_ptr++;
        current_green_thread->frames[current_green_thread->frame_ptr].func_name = fn;
        current_green_thread->frames[current_green_thread->frame_ptr].file = file;
        current_green_thread->frames[current_green_thread->frame_ptr].line = line;
    }
}

void kv_pop_frame() {
    if (current_green_thread && current_green_thread->frame_ptr >= 0) current_green_thread->frame_ptr--;
}

void kv_backtrace() {
    if (!current_green_thread) return;
    printf("--- KV Backtrace ---\n");
    for (int i = current_green_thread->frame_ptr; i >= 0; i--) {
        printf("  at %s (%s:%d)\n", current_green_thread->frames[i].func_name, current_green_thread->frames[i].file, current_green_thread->frames[i].line);
    }
}

#define KV_ENTER_FRAME(fn, file, line) kv_push_frame(fn, file, line);
#define KV_RECOVERY_POINT() if (setjmp(current_green_thread->frames[current_green_thread->frame_ptr].env) != 0) { VoltValue* _err = make_err(make_str("KV Runtime Error")); kv_pop_frame(); return _err; }
#define KV_RECOVERY_POINT_VOID() if (setjmp(current_green_thread->frames[current_green_thread->frame_ptr].env) != 0) { kv_pop_frame(); return; }
#define KV_EXIT_FRAME() kv_pop_frame();

void render_yield(VoltBuffer* ctx) {
    if (current_green_thread && current_green_thread->current_yield) {
        current_green_thread->current_yield(ctx);
    }
}

void volt_plugin_register(const char* path, struct Router* r) {
    void* handle = dlopen(path, RTLD_NOW);
    if (!handle) { printf("Plugin error: %s\n", dlerror()); return; }
    void (*init)(struct Router*) = dlsym(handle, "volt_plugin_init");
    if (init) init(r);
}

pthread_mutex_t db_file_lock = PTHREAD_MUTEX_INITIALIZER;

int global_epoll_fd;

void volt_netpoller_init() {
    global_epoll_fd = epoll_create1(0);
}

void volt_scheduler_yield() {
    if (current_green_thread) {
        swapcontext(&current_green_thread->context, &processors[worker_id].worker_context);
    }
}

void volt_poller_loop(void* arg) {
    struct epoll_event events[64];
    while (1) {
        int nfds = epoll_wait(global_epoll_fd, events, 64, 10);
        for (int i = 0; i < nfds; i++) {
            VoltGreenThread* t = (VoltGreenThread*)events[i].data.ptr;
            t->state = THREAD_READY;
            enqueue_thread(rand() % NUM_WORKERS, t);
        }
    }
}

typedef struct {
    int client_fd;
    char* method;
    char* path;
    char* headers[50];
    int header_count;
    int status;
    bool is_fragment;
    VoltBuffer* response_body;
    char* request_body;
    void** allocations;
    int alloc_count;
    int alloc_cap;
} VoltContext;

__thread VoltContext* current_web_ctx = NULL;

void* volt_track_alloc_raw(void* ptr) {
    if (current_web_ctx && ptr) {
        if (current_web_ctx->alloc_count >= current_web_ctx->alloc_cap) {
            current_web_ctx->alloc_cap = (current_web_ctx->alloc_cap == 0) ? 128 : current_web_ctx->alloc_cap * 2;
            current_web_ctx->allocations = realloc(current_web_ctx->allocations, current_web_ctx->alloc_cap * sizeof(void*));
        }
        current_web_ctx->allocations[current_web_ctx->alloc_count++] = ptr;
    }
    return ptr;
}

void* volt_track_alloc(size_t size) {
    void* ptr = malloc(size);
    return volt_track_alloc_raw(ptr);
}

void volt_green_thread_entry();

void enqueue_thread(int p_id, VoltGreenThread* t) {
    Processor* p = &processors[p_id];
    pthread_mutex_lock(&p->lock);
    int next = (p->tail + 1) % MAX_TASKS;
    if (next != p->head) {
        p->queue[p->tail] = t;
        p->tail = next;
    }
    pthread_mutex_unlock(&p->lock);
}

void schedule_task(int p_id, void (*func)(void*), void* arg) {
    VoltGreenThread* t = malloc(sizeof(VoltGreenThread));
    t->func = func;
    t->arg = arg;
    t->state = THREAD_READY;
    t->fd = -1;
    t->stack = malloc(STACK_SIZE);

    getcontext(&t->context);
    t->context.uc_stack.ss_sp = t->stack;
    t->context.uc_stack.ss_size = STACK_SIZE;
    t->context.uc_link = NULL;
    makecontext(&t->context, (void(*)())volt_green_thread_entry, 1, t);

    enqueue_thread(p_id, t);
}

void* worker_loop(void* arg) {
    worker_id = *(int*)arg;
    while (1) {
        Processor* p = &processors[worker_id];
        VoltGreenThread* t = NULL;
        pthread_mutex_lock(&p->lock);
        if (p->head != p->tail) {
            t = p->queue[p->head];
            p->head = (p->head + 1) % MAX_TASKS;
        }
        pthread_mutex_unlock(&p->lock);

        if (t) {
            t->state = THREAD_RUNNING;
            current_green_thread = t;
            swapcontext(&p->worker_context, &t->context);
            current_green_thread = NULL;
            if (t->state == THREAD_FINISHED) {
                free(t->stack);
                free(t);
            }
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
                if (t) {
                    t->state = THREAD_RUNNING;
                    current_green_thread = t;
                    swapcontext(&p->worker_context, &t->context);
                    current_green_thread = NULL;
                    if (t->state == THREAD_FINISHED) {
                        free(t->stack);
                        free(t);
                    }
                }
            }
            usleep(1000);
        }
    }
    return NULL;
}

void volt_green_thread_entry(VoltGreenThread* t) {
    t->func(t->arg);
    t->state = THREAD_FINISHED;
    setcontext(&processors[worker_id].worker_context);
}

// Runtime Primitives
VoltValue* make_str(const char* s) {
    VoltValue* v = volt_track_alloc(sizeof(VoltValue));
    v->type = TYPE_STR; v->s = strdup(s ? s : "");
    if (current_web_ctx) volt_track_alloc_raw(v->s);
    return v;
}
VoltValue* make_int(long long i) {
    VoltValue* v = volt_track_alloc(sizeof(VoltValue));
    v->type = TYPE_INT; v->i = i; return v;
}
VoltValue* make_bool(bool b) {
    VoltValue* v = volt_track_alloc(sizeof(VoltValue));
    v->type = TYPE_BOOL; v->b = b; return v;
}
VoltValue* make_fn(VoltFn f) {
    VoltValue* v = volt_track_alloc(sizeof(VoltValue));
    v->type = TYPE_FN; v->f = f; return v;
}

VoltValue* make_ok(VoltValue* v) {
    VoltValue* rv = volt_track_alloc(sizeof(VoltValue));
    rv->type = TYPE_RESULT; rv->res.is_ok = true; rv->res.val = v; return rv;
}

VoltValue* make_err(VoltValue* v) {
    VoltValue* rv = volt_track_alloc(sizeof(VoltValue));
    rv->type = TYPE_RESULT; rv->res.is_ok = false; rv->res.val = v; return rv;
}

VoltValue* volt_value_copy(VoltValue* v) {
    if (!v) return NULL;
    VoltValue* res = volt_track_alloc(sizeof(VoltValue));
    res->type = v->type;
    switch(v->type) {
        case TYPE_STR:
            res->s = strdup(v->s);
            if (current_web_ctx) volt_track_alloc_raw(res->s);
            break;
        case TYPE_INT: res->i = v->i; break;
        case TYPE_BOOL: res->b = v->b; break;
        case TYPE_FN: res->f = v->f; break;
        case TYPE_ARRAY:
            res->a.len = v->a.len;
            res->a.elements = malloc(v->a.len * sizeof(VoltValue*));
            if (current_web_ctx) volt_track_alloc_raw(res->a.elements);
            for(int i=0; i<v->a.len; i++) res->a.elements[i] = volt_value_copy(v->a.elements[i]);
            break;
        case TYPE_MAP:
            res->m.len = v->m.len;
            res->m.keys = malloc(v->m.len * sizeof(char*));
            res->m.values = malloc(v->m.len * sizeof(VoltValue*));
            if (current_web_ctx) {
                volt_track_alloc_raw(res->m.keys);
                volt_track_alloc_raw(res->m.values);
            }
            for(int i=0; i<v->m.len; i++) {
                res->m.keys[i] = strdup(v->m.keys[i]);
                if (current_web_ctx) volt_track_alloc_raw(res->m.keys[i]);
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

void volt_buf_append_json(VoltBuffer* b, VoltValue* v) {
    if (!v) { volt_buf_append(b, "null"); return; }
    char buf[128];
    switch(v->type) {
        case TYPE_STR:
            volt_buf_append(b, "\"");
            volt_buf_append(b, v->s);
            volt_buf_append(b, "\"");
            break;
        case TYPE_INT:
            sprintf(buf, "%lld", v->i);
            volt_buf_append(b, buf);
            break;
        case TYPE_BOOL:
            volt_buf_append(b, v->b ? "true" : "false");
            break;
        case TYPE_ARRAY:
            volt_buf_append(b, "[");
            for(int i=0; i<v->a.len; i++) {
                volt_buf_append_json(b, v->a.elements[i]);
                if (i < v->a.len - 1) volt_buf_append(b, ",");
            }
            volt_buf_append(b, "]");
            break;
        case TYPE_MAP:
            volt_buf_append(b, "{");
            for(int i=0; i<v->m.len; i++) {
                volt_buf_append(b, "\"");
                volt_buf_append(b, v->m.keys[i]);
                volt_buf_append(b, "\":");
                volt_buf_append_json(b, v->m.values[i]);
                if (i < v->m.len - 1) volt_buf_append(b, ",");
            }
            volt_buf_append(b, "}");
            break;
        default:
            volt_buf_append(b, "null");
    }
}

void volt_buf_append_escaped(VoltBuffer* b, const char* s) {
    if (!s) return;
    for (int i = 0; s[i]; i++) {
        switch(s[i]) {
            case '&': volt_buf_append(b, "&amp;"); break;
            case '<': volt_buf_append(b, "&lt;"); break;
            case '>': volt_buf_append(b, "&gt;"); break;
            case '"': volt_buf_append(b, "&quot;"); break;
            case '\'': volt_buf_append(b, "&#39;"); break;
            default: {
                char buf[2] = {s[i], 0};
                volt_buf_append(b, buf);
            }
        }
    }
}

void volt_buf_append_value(VoltBuffer* b, VoltValue* v) {
    char buf[128];
    if (v->type == TYPE_STR) {
        volt_buf_append_escaped(b, v->s);
    } else {
        volt_buf_append(b, to_str_buf(v, buf));
    }
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
    VoltValue* m = volt_track_alloc(sizeof(VoltValue));
    m->type = TYPE_MAP; m->m.len = 0;
    m->m.keys = malloc(20 * sizeof(char*));
    m->m.values = malloc(20 * sizeof(VoltValue*));
    if (current_web_ctx) {
        volt_track_alloc_raw(m->m.keys);
        volt_track_alloc_raw(m->m.values);
    }
    char* s = strdup(json);
    char* saveptr;
    char* p = strtok_r(s, "{}\",: ", &saveptr);
    while(p && m->m.len < 20) {
        m->m.keys[m->m.len] = strdup(p);
        if (current_web_ctx) volt_track_alloc_raw(m->m.keys[m->m.len]);
        p = strtok_r(NULL, "{}\",: ", &saveptr);
        if (p) {
            if (isdigit(*p)) m->m.values[m->m.len++] = make_int(atoll(p));
            else m->m.values[m->m.len++] = make_str(p);
        }
        p = strtok_r(NULL, "{}\",: ", &saveptr);
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
    if (!f) {
        pthread_mutex_unlock(&db_file_lock);
        if (kv_frame_ptr >= 0) {
            kv_backtrace();
            longjmp(kv_frame_stack[kv_frame_ptr].env, 1);
        }
        return;
    }
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

// Platform Abstraction Layer (PAL)
#ifdef _WIN32
#include <windows.h>
#define KV_FS_REMOVE(p) DeleteFile(p)
#define KV_FS_RENAME(s, d) MoveFile(s, d)
#else
#define KV_FS_REMOVE(p) unlink(p)
#define KV_FS_RENAME(s, d) rename(s, d)
#endif

// FS Shortcuts
void fs_rm(const char* path) { KV_FS_REMOVE(path); }
void fs_mv(const char* src, const char* dst) { KV_FS_RENAME(src, dst); }
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
typedef struct Route { char* path; void (*handler)(struct VoltContext*); bool is_ws; } Route;
typedef struct Router { char* name; Route routes[100]; int count; void (*before)(struct VoltContext*); } Router;

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
    volt_buf_append_json(current_web_ctx->response_body, v);
    return v;
}


VoltValue* volt_request_form(const char* key) {
    if (!current_web_ctx || !current_web_ctx->request_body) return make_str("");
    char* body_copy = strdup(current_web_ctx->request_body);
    char* saveptr1, *saveptr2;
    char* pair = strtok_r(body_copy, "&", &saveptr1);
    while(pair) {
        char* k = strtok_r(pair, "=", &saveptr2);
        char* v = strtok_r(NULL, "=", &saveptr2);
        if (k && v && strcmp(k, key) == 0) {
            VoltValue* rv = make_str(v);
            free(body_copy);
            return rv;
        }
        pair = strtok_r(NULL, "&", &saveptr1);
    }
    free(body_copy);
    return make_str("");
}

VoltValue* volt_request_json() {
    if (!current_web_ctx || !current_web_ctx->request_body) return json_parse("{}");
    return json_parse(current_web_ctx->request_body);
}

typedef struct { Router* r; int client_fd; } ConnTaskArgs;

void volt_dispatch_route(void* arg) {
    ConnTaskArgs* args = (ConnTaskArgs*)arg;
    VoltContext* ctx = malloc(sizeof(VoltContext));
    ctx->client_fd = args->client_fd;
    ctx->status = 200;
    ctx->is_fragment = false;
    ctx->header_count = 0;
    ctx->alloc_count = 0;
    ctx->alloc_cap = 128;
    ctx->allocations = malloc(ctx->alloc_cap * sizeof(void*));
    ctx->response_body = volt_buf_new();
    ctx->request_body = NULL;
    ctx->method = NULL;
    ctx->path = NULL;
    current_web_ctx = ctx;

    int flags = fcntl(ctx->client_fd, F_GETFL, 0);
    fcntl(ctx->client_fd, F_SETFL, flags | O_NONBLOCK);
    current_green_thread->fd = ctx->client_fd;

    char buffer[4096];
    int n;
    while ((n = read(ctx->client_fd, buffer, 4095)) < 0) {
        if (errno == EAGAIN || errno == EWOULDBLOCK) {
            struct epoll_event ev = { .events = EPOLLIN | EPOLLONESHOT, .data.ptr = current_green_thread };
            epoll_ctl(global_epoll_fd, EPOLL_CTL_ADD, ctx->client_fd, &ev);
            current_green_thread->state = THREAD_PARKED;
            volt_scheduler_yield();
        } else break;
    }

    if (n > 0) {
        buffer[n] = '\0';
        char* body_ptr = strstr(buffer, "\r\n\r\n");
        if (body_ptr) {
            *body_ptr = '\0';
            ctx->request_body = strdup(body_ptr + 4);
        }
        char* saveptr;
        char* method_str = strtok_r(buffer, " ", &saveptr);
        char* path_str = strtok_r(NULL, " ", &saveptr);
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
                return; // Maintain FD for spawned tasks
            } else {
                target->handler(ctx);
                fflush(stdout);
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
    if (ctx->request_body) free(ctx->request_body);
    if (ctx->method) free(ctx->method);
    if (ctx->path) free(ctx->path);
    for(int i=0; i<ctx->header_count; i++) free(ctx->headers[i]);
    for(int i=0; i<ctx->alloc_count; i++) free(ctx->allocations[i]);
    if (ctx->allocations) free(ctx->allocations);
    free(ctx);
    free(args);
}

void* volt_accept_loop(void* arg) {
    Router* r = (Router*)arg;
    int server_fd = socket(AF_INET, SOCK_STREAM, 0);
    struct sockaddr_in addr = { .sin_family = AF_INET, .sin_addr.s_addr = INADDR_ANY, .sin_port = htons(8080) };
    int opt = 1; setsockopt(server_fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));
    if (bind(server_fd, (struct sockaddr*)&addr, sizeof(addr)) < 0) {
        perror("bind failed");
        return NULL;
    }
    if (listen(server_fd, 100) < 0) {
        perror("listen failed");
        return NULL;
    }
    int flags = fcntl(server_fd, F_GETFL, 0);
    fcntl(server_fd, F_SETFL, flags | O_NONBLOCK);

    while(1) {
        int client = accept(server_fd, NULL, NULL);
        if (client < 0) {
            if (errno == EAGAIN || errno == EWOULDBLOCK) {
                // In a real netpoller, the accept loop would be a green thread too.
                // For now, we use a simple usleep to prevent busy waiting in this background thread.
                usleep(1000);
                continue;
            }
            perror("accept failed");
            continue;
        }
        ConnTaskArgs* args = malloc(sizeof(ConnTaskArgs));
        args->r = r; args->client_fd = client;
        schedule_task(rand() % NUM_WORKERS, volt_dispatch_route, args);
    }
    return NULL;
}

void volt_start_web_server(Router* r, int port) {
    volt_netpoller_init();
    pthread_t t1, t2;
    pthread_create(&t1, NULL, volt_accept_loop, r);
    pthread_create(&t2, NULL, (void* (*)(void*))volt_poller_loop, NULL);
    printf("KS-Panel Engine: Web server '%s' started on port %d with Netpoller\n", r->name, port);
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
void render_Card(VoltBuffer* ctx, int argc, VoltValue** argv) {
    KV_ENTER_FRAME("Card", "component", 0);
    KV_RECOVERY_POINT_VOID();
    volt_buf_append(ctx, "<div class=\"card\">");
    render_yield(ctx);
    volt_buf_append(ctx, "</div>");
    KV_EXIT_FRAME();
}
void render_Footer(VoltBuffer* ctx, int argc, VoltValue** argv) {
    KV_ENTER_FRAME("Footer", "component", 0);
    KV_RECOVERY_POINT_VOID();
    volt_buf_append(ctx, "<footer>Powered by KS-Volt</footer>");
    KV_EXIT_FRAME();
}
void render_Sidebar(VoltBuffer* ctx, int argc, VoltValue** argv) {
    KV_ENTER_FRAME("Sidebar", "component", 0);
    KV_RECOVERY_POINT_VOID();
    volt_buf_append(ctx, "<aside>Sidebar Content</aside>");
    KV_EXIT_FRAME();
}
void render_Navbar(VoltBuffer* ctx, int argc, VoltValue** argv) {
    KV_ENTER_FRAME("Navbar", "component", 0);
    KV_RECOVERY_POINT_VOID();
    volt_buf_append(ctx, "<nav>Navbar Content</nav>");
    KV_EXIT_FRAME();
}
Router router_main_app = { .name = "main_app", .count = 0 };
void handler_auto_dashboard_0(VoltContext* ctx) {
    KV_ENTER_FRAME("auto_handler", "web", 0);
    KV_RECOVERY_POINT_VOID();
    volt_value_free(({ VoltValue* _rv = NULL; VoltValue** _argv = NULL; render_Navbar(current_web_ctx->response_body, 0, _argv); _rv; }));
    volt_value_free(({ VoltValue* _rv = NULL; VoltValue** _argv = NULL; render_Card(current_web_ctx->response_body, 0, _argv); _rv; }));
    volt_value_free(make_str(""));
    volt_buf_append(current_web_ctx->response_body, "<h1>XSS Test: ");
    volt_buf_append(current_web_ctx->response_body, "<script>alert(1)</script>");
    volt_buf_append(current_web_ctx->response_body, "</h1>");
    volt_buf_append(current_web_ctx->response_body, "<h2>Raw Test: ");
    { VoltValue* _tmp = ({ VoltValue* _v = make_str("<p>Safe HTML</p>"); if (_v->type == TYPE_STR) volt_buf_append(ctx, _v->s); volt_value_free(_v); make_str(""); }); volt_buf_append_value(current_web_ctx->response_body, _tmp); volt_value_free(_tmp); }
    volt_buf_append(current_web_ctx->response_body, "</h2>");
    volt_value_free(make_str(""));
    volt_value_free(({ VoltValue* _rv = NULL; VoltValue** _argv = NULL; render_Footer(current_web_ctx->response_body, 0, _argv); _rv; }));
    KV_EXIT_FRAME();
}
__attribute__((constructor)) void init_handler_auto_dashboard_0() { router_main_app.routes[router_main_app.count++] = (Route){"/dashboard", handler_auto_dashboard_0, false }; }
Router router_main_app = { .name = "main_app", .count = 0 };
void handler_1(VoltContext* ctx) {
    KV_ENTER_FRAME("handler", "web", 0);
    KV_RECOVERY_POINT_VOID();
    volt_buf_append(current_web_ctx->response_body, "Login Page");
    KV_EXIT_FRAME();
}
__attribute__((constructor)) void init_handler_1() { router_main_app.routes[router_main_app.count++] = (Route){"/login", handler_1, false }; }

int main(int argc, char** argv) {
    setvbuf(stdout, NULL, _IONBF, 0);
    srand(time(NULL));
    for(int i=0; i<NUM_WORKERS; i++) {
        int* id = malloc(sizeof(int)); *id = i;
        pthread_mutex_init(&processors[i].lock, NULL);
        pthread_create(&workers[i], NULL, worker_loop, id);
    }
    volt_start_web_server(&router_main_app, 8080);
    volt_start_web_server(&router_main_app, 8080);
    volt_value_free(({ VoltValue* _rv = NULL; VoltValue* _a0 = make_str("MyVoltApp Backend Initialized"); VoltValue** _argv = malloc(1 * sizeof(VoltValue*)); _argv[0] = _a0; printf("%s\n", to_str(_a0)); volt_value_free(_a0); free(_argv); _rv; }));
    while(1) sleep(1); return 0;
}
