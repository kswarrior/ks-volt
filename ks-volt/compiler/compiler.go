package compiler

import (
	"fmt"
	"ks-volt/ast"
	"strings"
)

type Compiler struct {
	declaredVars map[string]bool
}

func New() *Compiler {
	return &Compiler{
		declaredVars: make(map[string]bool),
	}
}

func (c *Compiler) Compile(program *ast.Program) string {
	var sb strings.Builder

	sb.WriteString("package main\n\n")
	sb.WriteString("import (\n")
	sb.WriteString("\t\"encoding/json\"\n")
	sb.WriteString("\t\"fmt\"\n")
	sb.WriteString("\t\"io\"\n")
	sb.WriteString("\t\"net\"\n")
	sb.WriteString("\t\"net/http\"\n")
	sb.WriteString("\t\"os\"\n")
	sb.WriteString("\t\"sync\"\n")
	sb.WriteString("\t\"time\"\n")
	sb.WriteString(")\n\n")

	sb.WriteString("const DB_FILE = \"volt_db.json\"\n\n")

	// Runtime helper functions
	sb.WriteString(`
var (
	dbMutex        sync.RWMutex
	wg             sync.WaitGroup
	eventsMutex    sync.RWMutex
	eventHandlers  = make(map[string][]func())
)

func dbSave(key, value string) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	db := make(map[string]string)
	data, err := os.ReadFile(DB_FILE)
	if err == nil {
		json.Unmarshal(data, &db)
	}

	db[key] = value
	newData, _ := json.MarshalIndent(db, "", "  ")
	os.WriteFile(DB_FILE, newData, 0644)
}

func dbGet(key string) string {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	db := make(map[string]string)
	data, err := os.ReadFile(DB_FILE)
	if err != nil {
		return ""
	}
	json.Unmarshal(data, &db)
	return db[key]
}

func fetchAPI(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

func fileWrite(filename, data string) {
	os.WriteFile(filename, []byte(data), 0644)
}

func onEvent(name string, handler func()) {
	eventsMutex.Lock()
	defer eventsMutex.Unlock()
	eventHandlers[name] = append(eventHandlers[name], handler)
}

func emitEvent(name string) {
	eventsMutex.RLock()
	defer eventsMutex.RUnlock()
	handlers := eventHandlers[name]
	for _, handler := range handlers {
		wg.Add(1)
		go func(h func()) {
			defer wg.Done()
			h()
		}(handler)
	}
}

func serveHTML(port interface{}, html string) {
	portStr := fmt.Sprintf(":%v", port)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, html)
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		http.ListenAndServe(portStr, mux)
	}()
}

func connectBot(server, port interface{}, body func()) {
	address := net.JoinHostPort(fmt.Sprintf("%v", server), fmt.Sprintf("%v", port))
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, err := net.DialTimeout("tcp", address, 5*time.Second)
		if err != nil {
			fmt.Printf("🤖 Bot failed to connect to %s: %s\n", address, err)
			return
		}
		conn.Close()
		body()
	}()
}

func runInterval(ms int64, body func()) {
	ticker := time.NewTicker(time.Duration(ms) * time.Millisecond)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range ticker.C {
			body()
		}
	}()
}
`)

	sb.WriteString("\nfunc main() {\n")
	for _, stmt := range program.Statements {
		sb.WriteString(c.compileStatement(stmt, "\t"))
	}
	sb.WriteString("\twg.Wait()\n")
	sb.WriteString("}\n")

	return sb.String()
}

func (c *Compiler) compileStatement(stmt ast.Statement, indent string) string {
	switch s := stmt.(type) {
	case *ast.AssignmentStatement:
		op := ":="
		if c.declaredVars[s.Name.Value] {
			op = "="
		} else {
			c.declaredVars[s.Name.Value] = true
		}
		return fmt.Sprintf("%s%s %s %s\n", indent, s.Name.Value, op, c.compileExpression(s.Value))
	case *ast.ExpressionStatement:
		return fmt.Sprintf("%s%s\n", indent, c.compileExpression(s.Expression))
	case *ast.SpawnStatement:
		switch s.Name.Value {
		case "connect_bot":
			return c.compileConnectBot(s, indent)
		case "interval":
			return c.compileInterval(s, indent)
		case "on":
			return c.compileOn(s, indent)
		default:
			return c.compileSpawn(s, indent)
		}
	}
	return ""
}

func (c *Compiler) compileExpression(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.Identifier:
		return e.Value
	case *ast.IntegerLiteral:
		return fmt.Sprintf("%d", e.Value)
	case *ast.StringLiteral:
		return fmt.Sprintf("%q", e.Value)
	case *ast.InfixExpression:
		return fmt.Sprintf("fmt.Sprintf(\"%%v%%v\", %s, %s)", c.compileExpression(e.Left), c.compileExpression(e.Right))
	case *ast.CallExpression:
		return c.compileCall(e)
	}
	return ""
}

func (c *Compiler) compileCall(e *ast.CallExpression) string {
	name := ""
	if ident, ok := e.Function.(*ast.Identifier); ok {
		name = ident.Value
	}

	args := []string{}
	for _, arg := range e.Arguments {
		args = append(args, c.compileExpression(arg))
	}

	switch name {
	case "print":
		return fmt.Sprintf("fmt.Println(%s)", strings.Join(args, ", "))
	case "db_save":
		return fmt.Sprintf("dbSave(%s, %s)", args[0], args[1])
	case "db_get":
		return fmt.Sprintf("dbGet(%s)", args[0])
	case "fetch_api":
		return fmt.Sprintf("fetchAPI(%s)", args[0])
	case "serve_html":
		return fmt.Sprintf("serveHTML(%s, %s)", args[0], args[1])
	case "file_write":
		return fmt.Sprintf("fileWrite(%s, %s)", args[0], args[1])
	case "emit":
		return fmt.Sprintf("emitEvent(%s)", args[0])
	}
	return ""
}

func (c *Compiler) compileSpawn(s *ast.SpawnStatement, indent string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%swg.Add(1)\n", indent))
	sb.WriteString(fmt.Sprintf("%sgo func() {\n", indent))
	sb.WriteString(fmt.Sprintf("%s\tdefer wg.Done()\n", indent))
	for _, stmt := range s.Body.Statements {
		sb.WriteString(c.compileStatement(stmt, indent+"\t"))
	}
	sb.WriteString(fmt.Sprintf("%s}()\n", indent))
	return sb.String()
}

func (c *Compiler) compileConnectBot(s *ast.SpawnStatement, indent string) string {
	var sb strings.Builder
	server := c.compileExpression(s.Args[0])
	port := c.compileExpression(s.Args[1])

	sb.WriteString(fmt.Sprintf("%sconnectBot(%s, %s, func() {\n", indent, server, port))
	for _, stmt := range s.Body.Statements {
		sb.WriteString(c.compileStatement(stmt, indent+"\t"))
	}
	sb.WriteString(fmt.Sprintf("%s})\n", indent))
	return sb.String()
}

func (c *Compiler) compileInterval(s *ast.SpawnStatement, indent string) string {
	var sb strings.Builder
	ms := c.compileExpression(s.Args[0])
	sb.WriteString(fmt.Sprintf("%srunInterval(int64(%s), func() {\n", indent, ms))
	for _, stmt := range s.Body.Statements {
		sb.WriteString(c.compileStatement(stmt, indent+"\t"))
	}
	sb.WriteString(fmt.Sprintf("%s})\n", indent))
	return sb.String()
}

func (c *Compiler) compileOn(s *ast.SpawnStatement, indent string) string {
	var sb strings.Builder
	eventName := c.compileExpression(s.Args[0])
	sb.WriteString(fmt.Sprintf("%sonEvent(%s, func() {\n", indent, eventName))
	for _, stmt := range s.Body.Statements {
		sb.WriteString(c.compileStatement(stmt, indent+"\t"))
	}
	sb.WriteString(fmt.Sprintf("%s})\n", indent))
	return sb.String()
}
