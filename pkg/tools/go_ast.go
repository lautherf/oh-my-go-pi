package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

type GoDeclTool struct{}

func (t *GoDeclTool) Name() string { return "go_decl" }

type goDeclArgs struct {
	Path string `json:"path"`
	Name string `json:"name,omitempty"`
}

type declInfo struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Receiver string `json:"receiver,omitempty"`
	Start    int    `json:"start_line"`
	End      int    `json:"end_line"`
	Body     string `json:"body,omitempty"`
}

func (t *GoDeclTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args goDeclArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("go_decl: invalid args: %w", err)
	}
	if args.Path == "" {
		return "", fmt.Errorf("go_decl: path is required")
	}

	src, err := os.ReadFile(args.Path)
	if err != nil {
		return "", fmt.Errorf("go_decl: %w", err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, args.Path, src, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("go_decl: %w", err)
	}

	var decls []declInfo

	for _, d := range f.Decls {
		switch decl := d.(type) {
		case *ast.FuncDecl:
			info := declInfo{
				Name:  decl.Name.Name,
				Kind:  "func",
				Start: fset.Position(decl.Pos()).Line,
				End:   fset.Position(decl.End()).Line,
			}
			if decl.Recv != nil && len(decl.Recv.List) > 0 {
				info.Kind = "method"
				recv := decl.Recv.List[0]
				switch t := recv.Type.(type) {
				case *ast.Ident:
					info.Receiver = t.Name
				case *ast.StarExpr:
					if ident, ok := t.X.(*ast.Ident); ok {
						info.Receiver = "*" + ident.Name
					}
				}
			}
			decls = append(decls, info)

		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					info := declInfo{
						Name:  s.Name.Name,
						Kind:  "type",
						Start: fset.Position(s.Pos()).Line,
						End:   fset.Position(s.End()).Line,
					}
					switch s.Type.(type) {
					case *ast.StructType:
						info.Kind = "struct"
					case *ast.InterfaceType:
						info.Kind = "interface"
					}
					decls = append(decls, info)

				case *ast.ValueSpec:
					for _, name := range s.Names {
						info := declInfo{
							Name:  name.Name,
							Kind:  "var",
							Start: fset.Position(name.Pos()).Line,
							End:   fset.Position(s.End()).Line,
						}
						if decl.Tok.String() == "const" {
							info.Kind = "const"
						}
						decls = append(decls, info)
					}
				}
			}
		}
	}

	if args.Name != "" {
		for _, d := range decls {
			if d.Name == args.Name {
				body := extractNode(fset, fset.Position(f.Pos()).Line, src, d.Start, d.End)
				d.Body = body
				return formatDecl(d), nil
			}
		}
		return "", fmt.Errorf("go_decl: declaration %q not found in %s", args.Name, args.Path)
	}

	if len(decls) == 0 {
		return "no declarations found", nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Declarations in %s:\n\n", args.Path)
	for _, d := range decls {
		b.WriteString(formatDeclShort(d))
	}
	return b.String(), nil
}

func extractNode(fset *token.FileSet, fileStartLine int, src []byte, startLine, endLine int) string {
	lines := strings.Split(string(src), "\n")
	start := startLine - fileStartLine
	if start < 0 {
		start = 0
	}
	end := endLine - fileStartLine
	if end >= len(lines) {
		end = len(lines) - 1
	}
	return strings.Join(lines[start:end+1], "\n")
}

func formatDecl(d declInfo) string {
	var b strings.Builder
	fmt.Fprintf(&b, "name: %s\n", d.Name)
	fmt.Fprintf(&b, "kind: %s\n", d.Kind)
	if d.Receiver != "" {
		fmt.Fprintf(&b, "receiver: %s\n", d.Receiver)
	}
	fmt.Fprintf(&b, "lines: %d-%d\n", d.Start, d.End)
	fmt.Fprintf(&b, "---\n%s\n---\n", d.Body)
	return b.String()
}

func formatDeclShort(d declInfo) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  %s %s", d.Kind, d.Name)
	if d.Receiver != "" {
		fmt.Fprintf(&b, " (%s)", d.Receiver)
	}
	fmt.Fprintf(&b, "  lines %d-%d\n", d.Start, d.End)
	return b.String()
}
