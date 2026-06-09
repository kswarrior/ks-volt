package evaluator

import (
	"encoding/json"
	"fmt"
	"io"
	"ks-volt/ast"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

const DB_FILE = "volt_db.json"

type Environment struct {
	store map[string]interface{}
	mutex sync.RWMutex
}

func NewEnvironment() *Environment {
	return &Environment{store: make(map[string]interface{})}
}

func (e *Environment) Get(name string) (interface{}, bool) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	val, ok := e.store[name]
	return val, ok
}

func (e *Environment) Set(name string, val interface{}) interface{} {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.store[name] = val
	return val
}

type Evaluator struct {
	wg      *sync.WaitGroup
	db      map[string]string
	dbMutex sync.RWMutex
}

func New() *Evaluator {
	ev := &Evaluator{
		wg: &sync.WaitGroup{},
		db: make(map[string]string),
	}
	ev.loadDB()
	return ev
}

func (ev *Evaluator) loadDB() {
	ev.dbMutex.Lock()
	defer ev.dbMutex.Unlock()

	data, err := os.ReadFile(DB_FILE)
	if err != nil {
		return
	}
	json.Unmarshal(data, &ev.db)
}

func (ev *Evaluator) saveDB() {
	ev.dbMutex.RLock()
	defer ev.dbMutex.RUnlock()

	data, _ := json.MarshalIndent(ev.db, "", "  ")
	os.WriteFile(DB_FILE, data, 0644)
}

func (ev *Evaluator) Eval(node ast.Node, env *Environment) interface{} {
	switch n := node.(type) {
	case *ast.Program:
		return ev.evalStatements(n.Statements, env)

	case *ast.AssignmentStatement:
		val := ev.Eval(n.Value, env)
		env.Set(n.Name.Value, val)
		return val

	case *ast.ExpressionStatement:
		return ev.Eval(n.Expression, env)

	case *ast.BlockStatement:
		return ev.evalStatements(n.Statements, env)

	case *ast.SpawnStatement:
		if n.Name.Value == "connect_bot" {
			return ev.evalConnectBot(n, env)
		}
		ev.wg.Add(1)
		go func() {
			defer ev.wg.Done()
			ev.Eval(n.Body, env)
		}()
		return nil

	case *ast.InfixExpression:
		left := ev.Eval(n.Left, env)
		right := ev.Eval(n.Right, env)
		return ev.evalInfixExpression(n.Operator, left, right)

	case *ast.CallExpression:
		functionName := ""
		if ident, ok := n.Function.(*ast.Identifier); ok {
			functionName = ident.Value
		}

		args := []interface{}{}
		for _, arg := range n.Arguments {
			args = append(args, ev.Eval(arg, env))
		}

		return ev.applyBuiltin(functionName, args)

	case *ast.Identifier:
		if val, ok := env.Get(n.Value); ok {
			return val
		}
		return n.Value

	case *ast.IntegerLiteral:
		return n.Value

	case *ast.StringLiteral:
		return n.Value
	}

	return nil
}

func (ev *Evaluator) evalStatements(stmts []ast.Statement, env *Environment) interface{} {
	var result interface{}
	for _, stmt := range stmts {
		result = ev.Eval(stmt, env)
	}
	return result
}

func (ev *Evaluator) evalInfixExpression(operator string, left, right interface{}) interface{} {
	switch operator {
	case "+":
		return fmt.Sprintf("%v%v", left, right)
	default:
		return nil
	}
}

func (ev *Evaluator) evalConnectBot(n *ast.SpawnStatement, env *Environment) interface{} {
	if len(n.Args) != 2 {
		fmt.Println("connect_bot requires 2 arguments: server and port")
		return nil
	}

	server := fmt.Sprintf("%v", ev.Eval(n.Args[0], env))
	port := fmt.Sprintf("%v", ev.Eval(n.Args[1], env))
	address := net.JoinHostPort(server, port)

	ev.wg.Add(1)
	go func() {
		defer ev.wg.Done()
		conn, err := net.DialTimeout("tcp", address, 5*time.Second)
		if err != nil {
			fmt.Printf("🤖 Bot failed to connect to %s: %s\n", address, err)
			return
		}
		conn.Close()
		ev.Eval(n.Body, env)
	}()

	return nil
}

func (ev *Evaluator) applyBuiltin(name string, args []interface{}) interface{} {
	switch name {
	case "print":
		fmt.Println(args...)
		return nil
	case "serve_html":
		if len(args) != 2 {
			return fmt.Errorf("serve_html requires 2 arguments: port and html content")
		}
		port := fmt.Sprintf(":%v", args[0])
		html := fmt.Sprintf("%v", args[1])

		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, html)
		})

		ev.wg.Add(1)
		go func() {
			defer ev.wg.Done()
			err := http.ListenAndServe(port, mux)
			if err != nil {
				fmt.Printf("Error starting server: %s\n", err)
			}
		}()
		return nil
	case "fetch_api":
		if len(args) != 1 {
			return fmt.Errorf("fetch_api requires 1 argument: url")
		}
		url := fmt.Sprintf("%v", args[0])
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Sprintf("Error fetching API: %s", err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Sprintf("Error reading API response: %s", err)
		}
		return string(body)
	case "db_save":
		if len(args) != 2 {
			return fmt.Errorf("db_save requires 2 arguments: key and value")
		}
		key := fmt.Sprintf("%v", args[0])
		val := fmt.Sprintf("%v", args[1])

		ev.dbMutex.Lock()
		ev.db[key] = val
		ev.dbMutex.Unlock()
		ev.saveDB()
		return nil
	case "db_get":
		if len(args) != 1 {
			return fmt.Errorf("db_get requires 1 argument: key")
		}
		key := fmt.Sprintf("%v", args[0])
		ev.dbMutex.RLock()
		val, ok := ev.db[key]
		ev.dbMutex.RUnlock()
		if !ok {
			return ""
		}
		return val
	default:
		return nil
	}
}

func (ev *Evaluator) Wait() {
	ev.wg.Wait()
}
