package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var sqlPattern = regexp.MustCompile(`(?i)(\bselect\b|\binsert\s+into\b|\bupdate\s+[a-z_][a-z0-9_]*\s+set\b|\bdelete\s+from\b|\bwith\s+(recursive\s+)?[a-z_][a-z0-9_]*\s+as\s*\(|\bcreate\s+(table|index|unique|function|trigger)\b|\balter\s+table\b|\bdrop\s+table\b)`)

type violation struct {
	position token.Position
	reason   string
}

func main() {
	root := flag.String("root", ".", "module root to scan")
	flag.Parse()

	violations, err := scan(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "raw SQL check failed:", err)
		os.Exit(2)
	}
	if len(violations) > 0 {
		for _, item := range violations {
			fmt.Fprintf(os.Stderr, "%s: %s\n", item.position, item.reason)
		}
		os.Exit(1)
	}
	fmt.Println("No raw SQL or direct query calls outside SQLC and the DDL migration runner.")
}

func scan(root string) ([]violation, error) {
	var found []violation
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		normalized := filepath.ToSlash(relative)
		if entry.IsDir() {
			if entry.Name() == "node_modules" || entry.Name() == ".git" {
				return filepath.SkipDir
			}
			if normalized == "internal/db/sqlc" || strings.HasPrefix(normalized, "internal/db/sqlc/") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || normalized == "cmd/checkrawsql/main.go" {
			return nil
		}
		if !strings.HasPrefix(normalized, "internal/") && !strings.HasPrefix(normalized, "cmd/") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		allowDDLExec := normalized == "internal/migrations/migrate.go"
		ast.Inspect(file, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.BasicLit:
				if typed.Kind == token.STRING && sqlPattern.MatchString(unquote(typed.Value)) {
					found = append(found, violation{
						position: fset.Position(typed.Pos()),
						reason:   "SQL string literal outside SQLC-generated code",
					})
				}
			case *ast.CallExpr:
				selector, ok := typed.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				switch selector.Sel.Name {
				case "QueryRow":
					found = append(found, violation{position: fset.Position(typed.Pos()), reason: "direct QueryRow call outside SQLC"})
				case "Query":
					if inner, ok := selector.X.(*ast.SelectorExpr); ok && inner.Sel.Name == "URL" {
						return true
					}
					found = append(found, violation{position: fset.Position(typed.Pos()), reason: "direct Query call outside SQLC"})
				case "Exec":
					if !allowDDLExec {
						found = append(found, violation{position: fset.Position(typed.Pos()), reason: "direct Exec call outside the DDL migration runner"})
					}
				}
			}
			return true
		})
		return nil
	})
	return found, err
}

func unquote(value string) string {
	if len(value) < 2 {
		return value
	}
	return value[1 : len(value)-1]
}
