package main

import (
	"os"
	"go/ast"
	"fmt"
)

func main() {
	yyDebug = 100
	yyErrorVerbose = true
	s := new(Scanner)
	l := new(Lexer)
	l.s = s

	l.s.Init(os.Stdin)
	yyParse(l)
	ast.Print(nil, l.result)
	Inspect(Stmts(l.result), visitor)
}

type inspector func(Node) bool
type Visitor func(Node) Visitor

func (f inspector) visit(node Node) Visitor {
	if f(node) {
		return f.visit
	}
	return nil
}

func Inspect(node Node, f func(Node) bool) {
	Walk(inspector(f).visit, node)
}

func Walk(v Visitor, node Node) {
	if node == nil {
		return
	}

	f := v(node)
	switch n := node.(type) {
	case Exprs:
		for _, e := range n {
			Walk(f, e)
		}
	case Stmts:
		for _, s := range n {
			Walk(f, s)
		}
	case CmdStmt:
		Walk(f, n.Cmd)
		walkExprs(f, Exprs(n.Args))
	case BeginStmt:
		Walk(f, Stmts(n.Body))
	case IfStmt:
		Walk(f, n.Cond)
		walkStmts(f, Stmts(n.Body))
		walkStmts(f, Stmts(n.Else))
	case FunctionStmt:
		walkExprs(f, Exprs(n.Args))
		walkStmts(f, Stmts(n.Body))
	case PipeStmt:
		Walk(f, n.Lhs)
		Walk(f, n.Rhs)
	case RedirectStmt:
		Walk(f, n.Lhs)
		Walk(f, n.Rhs)
	}
	v(nil)
}

func walkStmts(f Visitor, stmts Stmts) {
	if stmts != nil && len(stmts) > 0 {
		Walk(f, stmts)
	}
}

func walkExprs(f Visitor, exprs Exprs) {
	if exprs != nil && len(exprs) > 0 {
		Walk(f, exprs)
	}
}
