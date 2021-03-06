package yyparse

import (
	"fmt"
	"strconv"
	"text/scanner"

	"github.com/acomagu/parsefish/ast"
)

var (
	singleQuotedSpecials = []rune{-1, '\\', '\''}
	doubleQuotedSpecials = []rune{-1, '\\', '"', '$'}
	specials             = []rune{-1, '\t', '\n', '$', '?', '*', '~', '#', '(', ')', '{', '}', '[', ']', '<', '>', '^', '&', ';', '\'', '"', '\\', ' '}
)

type ExpectIdentError error
type ExpectVarNameError error

var keywords = map[int]string{
	IF:       "if",
	ELSE:     "else",
	BEGIN:    "begin",
	END:      "end",
	FUNCTION: "function",
}

type Scanner struct {
	scanner.Scanner
}

func newIdent(pos scanner.Position, name string) ast.Ident {
	ident := ast.Ident{Name: name}
	ident.SetPos(pos)
	return ident
}

func newVar(pos scanner.Position, name string) ast.VarExpr {
	v := ast.VarExpr{Name: name}
	v.SetPos(pos)
	return v
}

func newFD(pos scanner.Position, n int) ast.FD {
	fd := ast.FD{N: n}
	fd.SetPos(pos)
	return fd
}

func (s *Scanner) scanSingleQuoted() (ast.StrExpr, error) {
	str := ast.StrExpr{}
	s.Next() // The starting '\''

	for {
		switch {
		case s.Peek() == scanner.EOF:
			return ast.StrExpr{}, fmt.Errorf("unexpected EOF")

		case s.isSingleQuotedIdentChar(s.Peek()):
			ident, err := s.scanSingleQuotedIdent()
			if err != nil {
				return ast.StrExpr{}, err
			}

			str = append(str, ident)

		case s.Peek() == '\'':
			s.Next()
			return str, nil

		default:
			ident, err := s.scanSingleQuotedIdent()
			if err != nil {
				return ast.StrExpr{}, err
			}

			str = append(str, ident)
		}
	}
}

func (s *Scanner) scanDoubleQuoted() (ast.StrExpr, error) {
	str := ast.StrExpr{}
	s.Next() // The starting '"'

	for {
		switch {
		case s.Peek() == scanner.EOF:
			return ast.StrExpr{}, fmt.Errorf("unexpected EOF")

		case s.isDoubleQuotedIdentChar(s.Peek()):
			ident, err := s.scanDoubleQuotedIdent()
			if err != nil {
				return ast.StrExpr{}, err
			}

			str = append(str, ident)

		case s.Peek() == '$':
			vare, err := s.scanVar()
			if err != nil {
				return ast.StrExpr{}, err
			}

			str = append(str, vare)

		case s.Peek() == '"':
			s.Next()
			return str, nil

		default:
			ident, err := s.scanDoubleQuotedIdent()
			if err != nil {
				return ast.StrExpr{}, err
			}

			str = append(str, ident)
		}
	}
}

func (s *Scanner) isSpecialChar(c rune) bool {
	for _, sc := range specials {
		if sc == c {
			return true
		}
	}
	return false
}

func (s *Scanner) isSingleQuotedIdentChar(c rune) bool {
	return !s.isSingleQuotedSpecialChar(c)
}

func (s *Scanner) isDoubleQuotedIdentChar(c rune) bool {
	return !s.isDoubleQuotedSpecialChar(c)
}

func (s *Scanner) isDoubleQuotedSpecialChar(c rune) bool {
	for _, sc := range doubleQuotedSpecials {
		if sc == c {
			return true
		}
	}
	return false
}

func (s *Scanner) isSingleQuotedSpecialChar(c rune) bool {
	for _, sc := range singleQuotedSpecials {
		if sc == c {
			return true
		}
	}
	return false
}

func (s *Scanner) scanDoubleQuotedIdent() (ast.Ident, error) {
	pos := s.Pos()
	var ret []rune
	for s.isDoubleQuotedIdentChar(s.Peek()) {
		ret = append(ret, s.Peek())
		s.Next()
	}
	if len(ret) == 0 {
		return ast.Ident{}, ExpectIdentError(fmt.Errorf("expected IDENT"))
	}
	return newIdent(pos, string(ret)), nil
}

func (s *Scanner) scanSingleQuotedIdent() (ast.Ident, error) {
	pos := s.Pos()
	var ret []rune
	for s.isSingleQuotedIdentChar(s.Peek()) {
		ret = append(ret, s.Peek())
		s.Next()
	}
	if len(ret) == 0 {
		return ast.Ident{}, ExpectIdentError(fmt.Errorf("expected IDENT"))
	}
	return newIdent(pos, string(ret)), nil
}

func (s *Scanner) isNumber(c rune) bool {
	return '0' <= c && c <= '9'
}

func (s *Scanner) isIdentChar(c rune) bool {
	return !s.isSpecialChar(c)
}

func (s *Scanner) isLetter(c rune) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

func (s *Scanner) isBrank(c rune) bool {
	return c == ' ' || c == '\t'
}

func (s *Scanner) scanVar() (ast.VarExpr, error) {
	pos := s.Pos()

	s.Next() // The '$'
	var ret []rune
	for s.isVarChar(s.Peek()) {
		ret = append(ret, s.Peek())
		s.Next()
	}
	if len(ret) == 0 {
		return ast.VarExpr{}, ExpectVarNameError(fmt.Errorf("expected variable name after $"))
	}

	name := string(ret)
	return newVar(pos, name), nil
}

func (s *Scanner) isVarChar(c rune) bool {
	return s.isLetter(c) || s.isNumber(c) || c == '_'
}

func (s *Scanner) scanIdent() (ast.Ident, error) {
	pos := s.Pos()
	var ret []rune
	for s.isIdentChar(s.Peek()) {
		ret = append(ret, s.Peek())
		s.Next()
	}
	if len(ret) == 0 {
		return ast.Ident{}, ExpectIdentError(fmt.Errorf("expected IDENT"))
	}
	return newIdent(pos, string(ret)), nil
}

func (s *Scanner) isStrChar(c rune) bool {
	return true
}

func (s *Scanner) scanStrs() (int, ast.StrExpr, error) {
	withRightParen := false
	str := ast.StrExpr{}
	for {
		switch {
		case s.Peek() == '$':
			vare, err := s.scanVar()
			if err != nil {
				return 0, nil, err
			}
			str = append(str, vare)

		case s.Peek() == '"':
			quotedStr, err := s.scanDoubleQuoted()
			if err != nil {
				return 0, nil, err
			}

			str = append(str, quotedStr...)

		case s.Peek() == '\'':
			quotedStr, err := s.scanSingleQuoted()
			if err != nil {
				return 0, nil, err
			}

			str = append(str, quotedStr...)

		case s.Peek() == ')':
			if len(str) > 0 {
				return STR, str, nil
			}
			if withRightParen {
				if len(str) == 0 {
					return ')', str, nil
				}
				return RIGHT_PAREN_AND_STR, str, nil
			}
			s.Next()
			withRightParen = true

		case s.Peek() == '(':
			s.Next()
			if len(str) == 0 && !withRightParen {
				return '(', str, nil
			}
			if withRightParen {
				return RIGHT_PAREN_AND_STR_AND_LEFT_PAREN, str, nil
			}
			return STR_AND_LEFT_PAREN, str, nil

		case s.isSpecialChar(s.Peek()):
			if withRightParen {
				return RIGHT_PAREN_AND_STR, str, nil
			}
			return STR, str, nil

		default:
			ident, err := s.scanIdent()
			if err != nil {
				return 0, nil, err
			}
			str = append(str, ident)
		}
	}
}

func (s *Scanner) skipBrank() {
	for s.isBrank(s.Peek()) {
		s.Next()
	}
}

func (s *Scanner) skipComment() {
	for s.Peek() != '\n' {
		s.Next()
	}
}

func (s *Scanner) scanFD() (ast.FD, error) {
	pos := s.Pos()

	s.Next() // '&'
	if s.Peek() == '-' {
		return newFD(pos, -1), nil
	}

	ret := []rune{}
	for s.isNumber(s.Peek()) {
		ret = append(ret, s.Peek())
		s.Next()
	}
	n, err := strconv.Atoi(string(ret))
	if err != nil {
		return ast.FD{}, err
	}
	return newFD(pos, n), nil
}

func (*Scanner) tokenOf(e ast.StrExpr) (int, bool) {
	lit := ""
	for _, v := range e {
		ident, ok := v.(ast.Ident)
		if !ok {
			return 0, false
		}
		lit += ident.Name
	}

	for tok, l := range keywords {
		if l == lit {
			return tok, true
		}
	}
	return 0, false
}

func newScanner() Scanner {
	return Scanner{}
}

type Lexer struct {
	s      *Scanner
	result []ast.Stmt
}

func (l *Lexer) mainLex(lval *yySymType) (int, error) {
	s := l.s

RETRY:
	s.skipBrank()
	c := s.Peek()

	switch c {
	case scanner.EOF:
		return int(c), nil

	case '#':
		s.skipComment()
		goto RETRY

	// ">", ">&N", ">>"
	case '>':
		s.Next()
		switch s.Peek() {
		case '&':
			fd, err := s.scanFD()
			if err != nil {
				return 0, err
			}
			lval.expr = ast.StrExpr{fd}
			return REDIRECT_TO_FD, nil
		case '>':
			return APPEND_REDIRECT, nil
		}
		return '>', nil

	// "^", "^&N", "^^"
	case '^':
		s.Next()
		switch s.Peek() {
		case '&':
			fd, err := s.scanFD()
			if err != nil {
				return 0, err
			}
			lval.expr = ast.StrExpr{fd}
			return ERR_REDIRECT_TO_FD, nil

		case '^':
			return APPEND_ERR_REDIRECT, nil
		}
		return '^', nil

	case ';', '\n', '|', '<':
		s.Next()
		return int(c), nil

	default:
		tok, v, err := s.scanStrs()
		if err != nil {
			return 0, err
		}
		if tok != STR {
			lval.str = v
			return tok, nil
		}

		if tok, ok := s.tokenOf(v); ok {
			return tok, nil
		}
		lval.str = v

		return STR, nil
	}
}

func (l *Lexer) Lex(lval *yySymType) int {
	tok, err := l.mainLex(lval)
	if err != nil {
		panic(err)
	}

	return tok
}

func (l *Lexer) Error(e string) {
	panic(e)
}
