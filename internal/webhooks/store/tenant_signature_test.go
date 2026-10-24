package store

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// noTenantDirective marks a storage method that legitimately takes no tenant.
// It must sit in the method's doc comment and be followed by a reason.
const noTenantDirective = "//sparrow:no-tenant"

// TestStorageMethodsTakeTenant enforces the tenant rule for this package:
// every method of every interface takes ctx context.Context first and
// tenantID uuid.UUID second, unless its doc comment carries
// //sparrow:no-tenant <reason>. A storage call that cannot forget its tenant
// is how tenants stay apart; see AGENTS.md (Code conventions).
func TestStorageMethodsTakeTenant(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var violations []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(".", name), nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		violations = append(violations, checkTenantSignatures(file)...)
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		t.Fatalf("storage interface methods must take (ctx context.Context, tenantID uuid.UUID, ...) "+
			"or carry %q above them:\n  %s", noTenantDirective+" <reason>", strings.Join(violations, "\n  "))
	}
}

// TestCheckTenantSignaturesReportsViolations guards the checker itself.
func TestCheckTenantSignaturesReportsViolations(t *testing.T) {
	src := `package x
type Repo interface {
	Good(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
	Grouped(ctx context.Context, tenantID, id uuid.UUID) error
	MissingTenant(ctx context.Context, id uuid.UUID) error
	NoCtx(tenantID uuid.UUID) error
	WrongName(ctx context.Context, webhookID uuid.UUID) error
	//sparrow:no-tenant spans every tenant on purpose
	Exempt(ctx context.Context) error
	//sparrow:no-tenant
	ExemptWithoutReason(ctx context.Context) error
}
`
	file, err := parser.ParseFile(token.NewFileSet(), "x.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, v := range checkTenantSignatures(file) {
		got[strings.SplitN(v, ":", 2)[0]] = true
	}
	want := []string{"Repo.MissingTenant", "Repo.NoCtx", "Repo.WrongName", "Repo.ExemptWithoutReason"}
	for _, w := range want {
		if !got[w] {
			t.Errorf("expected a violation for %s, got %v", w, got)
		}
	}
	for _, ok := range []string{"Repo.Good", "Repo.Grouped", "Repo.Exempt"} {
		if got[ok] {
			t.Errorf("unexpected violation for %s", ok)
		}
	}
}

// checkTenantSignatures returns one "Interface.Method: problem" line per
// interface method in file that breaks the tenant rule.
func checkTenantSignatures(file *ast.File) []string {
	var out []string
	ast.Inspect(file, func(n ast.Node) bool {
		spec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		iface, ok := spec.Type.(*ast.InterfaceType)
		if !ok {
			return true
		}
		for _, field := range iface.Methods.List {
			fn, ok := field.Type.(*ast.FuncType)
			if !ok || len(field.Names) == 0 {
				continue // embedded interface
			}
			name := spec.Name.Name + "." + field.Names[0].Name
			if reason, exempt := exemption(field.Doc); exempt {
				if reason == "" {
					out = append(out, fmt.Sprintf("%s: %s needs a reason", name, noTenantDirective))
				}
				continue
			}
			if problem := tenantParamProblem(fn); problem != "" {
				out = append(out, name+": "+problem)
			}
		}
		return true
	})
	return out
}

// exemption reports whether doc carries the no-tenant directive, and the
// reason written after it.
func exemption(doc *ast.CommentGroup) (reason string, ok bool) {
	if doc == nil {
		return "", false
	}
	for _, c := range doc.List {
		if rest, found := strings.CutPrefix(c.Text, noTenantDirective); found {
			return strings.TrimSpace(rest), true
		}
	}
	return "", false
}

// tenantParamProblem describes how fn's first two parameters differ from
// (ctx context.Context, tenantID uuid.UUID), or returns "".
func tenantParamProblem(fn *ast.FuncType) string {
	type param struct{ name, typ string }
	var params []param
	for _, f := range fn.Params.List {
		typ := exprString(f.Type)
		if len(f.Names) == 0 {
			params = append(params, param{"", typ})
		}
		for _, n := range f.Names {
			params = append(params, param{n.Name, typ})
		}
	}
	if len(params) == 0 || params[0].typ != "context.Context" {
		return "first parameter is not context.Context"
	}
	if len(params) < 2 {
		return "no tenantID parameter"
	}
	if params[1].name != "tenantID" || params[1].typ != "uuid.UUID" {
		return fmt.Sprintf("second parameter is %s %s, not tenantID uuid.UUID", params[1].name, params[1].typ)
	}
	return ""
}

func exprString(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return exprString(v.X) + "." + v.Sel.Name
	case *ast.StarExpr:
		return "*" + exprString(v.X)
	case *ast.ArrayType:
		return "[]" + exprString(v.Elt)
	case *ast.FuncType:
		return "func"
	default:
		return fmt.Sprintf("%T", e)
	}
}

// TestCheckTenantSignaturesEdgeCases covers the Go syntax the checker must
// read correctly: grouped and unnamed parameters, variadic and func-typed
// parameters, embedded interfaces, multi-line doc comments, generic and
// non-interface types.
func TestCheckTenantSignaturesEdgeCases(t *testing.T) {
	src := `package x
type Embedded interface{ Good(ctx context.Context, tenantID uuid.UUID) error }
type Repo interface {
	Embedded
	GroupedAll(ctx context.Context, tenantID, a, b uuid.UUID) error
	Variadic(ctx context.Context, tenantID uuid.UUID, ids ...uuid.UUID) error
	FuncParam(ctx context.Context, tenantID uuid.UUID, fn func(int) error) error
	PointerTenant(ctx context.Context, tenantID *uuid.UUID) error
	Unnamed(context.Context, uuid.UUID) error
	SwappedOrder(tenantID uuid.UUID, ctx context.Context) error
	StringTenant(ctx context.Context, tenantID string) error
	// Exempt is documented over several lines,
	// and the directive is not on the first one.
	//sparrow:no-tenant spans every tenant
	Exempt(ctx context.Context) error
}
type NotAnInterface struct{ ctx context.Context }
type Generic[T any] interface {
	Get(ctx context.Context, tenantID uuid.UUID, v T) error
	Bad(ctx context.Context, v T) error
}
`
	file, err := parser.ParseFile(token.NewFileSet(), "x.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, v := range checkTenantSignatures(file) {
		got[strings.SplitN(v, ":", 2)[0]] = true
	}
	for _, bad := range []string{"Repo.PointerTenant", "Repo.Unnamed", "Repo.SwappedOrder", "Repo.StringTenant", "Generic.Bad"} {
		if !got[bad] {
			t.Errorf("%s should be a violation; got %v", bad, got)
		}
	}
	for _, ok := range []string{"Embedded.Good", "Repo.GroupedAll", "Repo.Variadic", "Repo.FuncParam", "Repo.Exempt", "Generic.Get", "Repo.Embedded"} {
		if got[ok] {
			t.Errorf("%s should not be a violation", ok)
		}
	}
	if len(got) != 5 {
		t.Errorf("want exactly 5 violations, got %v", got)
	}
}
