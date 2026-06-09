package evaluator

import (
	"fmt"
	"ks-volt/ast"
	"net/http"
	"sync"
)

type Environment struct {
	store map[string]interface{}
}

func NewEnvironment() *Environment {
	return &Environment{store: make(map[string]interface{})}
}

func (e *Environment) Get(name string) (interface{}, bool) {
	val, ok := e.store[name]
	return val, ok
}

func (e *Environment) Set(name string, val interface{}) interface{} {
	e.store[name] = val
	return val
}

type Evaluator struct {
	wg *sync.WaitGroup
}

func New() *Evaluator {
	return &Evaluator{wg: &sync.WaitGroup{}}
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
		ev.wg.Add(1)
		go func() {
			defer ev.wg.Done()
			// We might want to pass arguments to the "function" if we had real functions,
			// but for now let's just evaluate the body.
			ev.Eval(n.Body, env)
		}()
		return nil

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
		return n.Value // Return literal name if not in env (for builtins/function names)

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

		// We need to be careful with starting multiple servers on same port or root,
		// but for KS-Volt MVP this is fine.
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, html)
		})

		// This is a blocking call in Go, so we should probably run it in a way
		// that doesn't block the WHOLE evaluator if it's called inside spawn.
		// Since it's usually called inside spawn in our example, it's fine.
		err := http.ListenAndServe(port, nil)
		if err != nil {
			fmt.Printf("Error starting server: %s\n", err)
		}
		return nil
	default:
		return nil
	}
}

func (ev *Evaluator) Wait() {
	ev.wg.Wait()
}
