package main

import (
	"fmt"
	"ks-volt/compiler"
	"ks-volt/lexer"
	"ks-volt/parser"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	c := compiler.New()
	goCode := c.Compile(program)

	// Use a filename that doesn't start with a dot to avoid "no Go files" errors
	// in some environments when running go build on a single file.
	tempFile := "compiled_volt_tmp.go"
	err = os.WriteFile(tempFile, []byte(goCode), 0644)
	if err != nil {
		fmt.Printf("Error writing temporary Go file: %s\n", err)
		os.Exit(1)
	}
	defer os.Remove(tempFile)

	outputBinary := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))

	cmd := exec.Command("go", "build", "-o", outputBinary, tempFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error compiling KS-Volt script: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully compiled %s to %s\n", filename, outputBinary)
}

func printParserErrors(errors []string) {
	fmt.Println("Parser errors:")
	for _, msg := range errors {
		fmt.Printf("\t%s\n", msg)
	}
}
