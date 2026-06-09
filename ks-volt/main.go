package main

import (
	"fmt"
	"ks-volt/evaluator"
	"ks-volt/lexer"
	"ks-volt/parser"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ks-volt <filename.kv>")
		os.Exit(1)
	}

	filename := os.Args[1]
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file %s: %s\n", filename, err)
		os.Exit(1)
	}

	l := lexer.New(string(data))
	p := parser.New(l)

	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		printParserErrors(p.Errors())
		os.Exit(1)
	}

	env := evaluator.NewEnvironment()
	ev := evaluator.New()
	ev.Eval(program, env)

	// Wait for any spawned goroutines
	ev.Wait()
}

func printParserErrors(errors []string) {
	fmt.Println("Parser errors:")
	for _, msg := range errors {
		fmt.Printf("\t%s\n", msg)
	}
}
