package evaluator

import (
	"fmt"
	"io"
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
		res := fmt.Sprintf("%v%v", left, right)
		return res
	default:
		return nil
	}
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

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, html)
		})

		err := http.ListenAndServe(port, nil)
		if err != nil {
			fmt.Printf("Error starting server: %s\n", err)
		}
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
	default:
		return nil
	}
}

func (ev *Evaluator) Wait() {
	ev.wg.Wait()
}
