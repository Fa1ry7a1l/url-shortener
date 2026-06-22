package main

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// NoOSExitAnalyzer prohibits direct os.Exit calls inside main.main.
//
// Calling os.Exit from main skips deferred functions. The analyzer encourages
// main to return normally and lets startup helpers report errors to it.
var NoOSExitAnalyzer = &analysis.Analyzer{
	Name: "noosexit",
	Doc:  "prohibits direct calls to os.Exit in the main function of package main",
	Run:  runNoOSExit,
}

func runNoOSExit(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		if ast.IsGenerated(file) {
			continue
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name.Name != "main" || fn.Body == nil {
				continue
			}

			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}

				callee, ok := calledFunction(pass, call.Fun)
				if !ok || callee.Pkg() == nil {
					return true
				}
				if callee.Pkg().Path() == "os" && callee.Name() == "Exit" {
					pass.Reportf(call.Pos(), "direct call to os.Exit in main is prohibited")
				}

				return true
			})
		}
	}

	return nil, nil
}

func calledFunction(pass *analysis.Pass, expr ast.Expr) (*types.Func, bool) {
	switch expr := expr.(type) {
	case *ast.ParenExpr:
		return calledFunction(pass, expr.X)
	case *ast.SelectorExpr:
		fn, ok := pass.TypesInfo.Uses[expr.Sel].(*types.Func)
		return fn, ok
	case *ast.Ident:
		fn, ok := pass.TypesInfo.Uses[expr].(*types.Func)
		return fn, ok
	default:
		return nil, false
	}
}
