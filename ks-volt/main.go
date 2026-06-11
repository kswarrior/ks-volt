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
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ks-volt <filename.kv> OR ks-volt watch <filename.kv>")
		os.Exit(1)
	}

	mode := "compile"
	filename := os.Args[1]

	if os.Args[1] == "watch" {
		if len(os.Args) < 3 {
			fmt.Println("Usage: ks-volt watch <filename.kv>")
			os.Exit(1)
		}
		mode = "watch"
		filename = os.Args[2]
	}

	if mode == "watch" {
		fmt.Printf("Watching %s for changes...\n", filename)
		var lastMod time.Time
		for {
			info, err := os.Stat(filename)
			if err == nil {
				if info.ModTime().After(lastMod) {
					fmt.Printf("Change detected. Recompiling...\n")
					runCompilation(filename)
					lastMod = info.ModTime()
				}
			}
			time.Sleep(1 * time.Second)
		}
	} else {
		runCompilation(filename)
	}
}

func runCompilation(filename string) {
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
	cCode := c.Compile(program)

	tempFile := "compiled_volt_tmp.c"
	err = os.WriteFile(tempFile, []byte(cCode), 0644)
	if err != nil {
		fmt.Printf("Error writing temporary C file: %s\n", err)
		os.Exit(1)
	}

	outputBinary := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))

	gccArgs := []string{"-O3", "-pthread", "-s", "-w", "-DCONFIG_VERSION=\"2024-01-13\"", "-D_GNU_SOURCE", tempFile}

	// Add JS dependency
	gccArgs = append(gccArgs, "deps/quickjs.c", "deps/libbf.c", "deps/libunicode.c", "deps/cutils.c", "deps/libregexp.c", "deps/quickjs-libc.c")

	// Add Polyglot libs
	for _, lib := range c.LinkLibs {
		gccArgs = append(gccArgs, lib)
	}

	// Add Python flags if needed
	if c.PythonNeeded {
		pyCflags, _ := exec.Command("python3-config", "--cflags").Output()
		pyLdflags, _ := exec.Command("python3-config", "--embed", "--ldflags").Output()
		for _, f := range strings.Fields(string(pyCflags)) {
			gccArgs = append(gccArgs, f)
		}
		for _, f := range strings.Fields(string(pyLdflags)) {
			gccArgs = append(gccArgs, f)
		}
	}

	gccArgs = append(gccArgs, "-lm", "-ldl", "-o", outputBinary)

	// GCC Orchestration
	cmd := exec.Command("gcc", gccArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error compiling KS-Volt script via GCC: %s\n", err)
		return
	}

	fmt.Printf("Successfully compiled %s to STATIC NATIVE BINARY: %s\n", filename, outputBinary)
}

func printParserErrors(errors []string) {
	fmt.Println("Parser errors:")
	for _, msg := range errors {
		fmt.Printf("\t%s\n", msg)
	}
}
