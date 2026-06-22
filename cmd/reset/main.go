// Command reset generates Reset methods for structures marked with
// "// generate:reset".
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

const (
	generatedFileName = "reset.gen.go"
	resetMarker       = "generate:reset"
)

type markedStruct struct {
	name  string
	named *types.Named
}

type packageGeneration struct {
	name    string
	dir     string
	types   *types.Package
	structs []markedStruct
}

func main() {
	root := flag.String("root", ".", "root directory to scan")
	flag.Parse()

	if err := generate(*root); err != nil {
		log.Fatal(err)
	}
}

func generate(root string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve root directory: %w", err)
	}

	overlay, existingGeneratedFiles, err := generatedFileOverlay(root)
	if err != nil {
		return err
	}

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo,
		Dir:     root,
		Overlay: overlay,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return fmt.Errorf("scan packages: %w", err)
	}
	if err := packageErrors(pkgs); err != nil {
		return err
	}

	generations := make([]packageGeneration, 0)
	loadedDirs := make(map[string]struct{})

	for _, pkg := range pkgs {
		dir, ok := packageDir(root, pkg)
		if !ok {
			continue
		}
		loadedDirs[dir] = struct{}{}

		structs, err := findMarkedStructs(pkg)
		if err != nil {
			return err
		}
		if len(structs) == 0 {
			continue
		}

		sort.Slice(structs, func(i, j int) bool {
			return structs[i].name < structs[j].name
		})
		generations = append(generations, packageGeneration{
			name:    pkg.Name,
			dir:     dir,
			types:   pkg.Types,
			structs: structs,
		})
	}

	sort.Slice(generations, func(i, j int) bool {
		return generations[i].dir < generations[j].dir
	})

	outputs := make(map[string][]byte, len(generations))
	for _, generation := range generations {
		data, err := renderPackage(generation)
		if err != nil {
			return fmt.Errorf("generate package %s: %w", generation.types.Path(), err)
		}
		outputs[filepath.Join(generation.dir, generatedFileName)] = data
	}

	for path, data := range outputs {
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}

	for _, path := range existingGeneratedFiles {
		if _, wanted := outputs[path]; wanted {
			continue
		}
		if _, loaded := loadedDirs[filepath.Dir(path)]; !loaded {
			continue
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove stale generated file %s: %w", path, err)
		}
	}

	return nil
}

func generatedFileOverlay(root string) (map[string][]byte, []string, error) {
	overlay := make(map[string][]byte)
	var paths []string

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "vendor":
				if path != root {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if entry.Name() != generatedFileName {
			return nil
		}

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
		if err != nil {
			return fmt.Errorf("parse generated file %s: %w", path, err)
		}
		absolutePath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("resolve generated file %s: %w", path, err)
		}

		overlay[absolutePath] = []byte("package " + file.Name.Name + "\n")
		paths = append(paths, absolutePath)
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("find generated files: %w", err)
	}

	return overlay, paths, nil
}

func packageErrors(pkgs []*packages.Package) error {
	var messages []string
	for _, pkg := range pkgs {
		for _, pkgErr := range pkg.Errors {
			messages = append(messages, pkgErr.Error())
		}
	}
	if len(messages) == 0 {
		return nil
	}
	sort.Strings(messages)
	return fmt.Errorf("load packages:\n%s", strings.Join(messages, "\n"))
}

func packageDir(root string, pkg *packages.Package) (string, bool) {
	files := pkg.GoFiles
	if len(files) == 0 {
		files = pkg.CompiledGoFiles
	}
	if len(files) == 0 {
		return "", false
	}

	dir := filepath.Dir(files[0])
	relative, err := filepath.Rel(root, dir)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}
	return dir, true
}

func findMarkedStructs(pkg *packages.Package) ([]markedStruct, error) {
	var result []markedStruct

	for _, file := range pkg.Syntax {
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE {
				continue
			}

			for _, specification := range general.Specs {
				typeSpec, ok := specification.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if !hasResetMarker(typeSpec.Doc) &&
					!(len(general.Specs) == 1 && hasResetMarker(general.Doc)) {
					continue
				}
				if _, ok := typeSpec.Type.(*ast.StructType); !ok {
					return nil, fmt.Errorf(
						"%s: // %s can only be used with a structure",
						pkg.Fset.Position(typeSpec.Pos()),
						resetMarker,
					)
				}
				if typeSpec.Assign.IsValid() {
					return nil, fmt.Errorf(
						"%s: cannot generate a method for a type alias",
						pkg.Fset.Position(typeSpec.Pos()),
					)
				}

				object := pkg.TypesInfo.Defs[typeSpec.Name]
				if object == nil {
					return nil, fmt.Errorf(
						"%s: cannot resolve type %s",
						pkg.Fset.Position(typeSpec.Pos()),
						typeSpec.Name.Name,
					)
				}
				named, ok := types.Unalias(object.Type()).(*types.Named)
				if !ok {
					return nil, fmt.Errorf(
						"%s: %s is not a defined structure type",
						pkg.Fset.Position(typeSpec.Pos()),
						typeSpec.Name.Name,
					)
				}

				result = append(result, markedStruct{
					name:  typeSpec.Name.Name,
					named: named,
				})
			}
		}
	}

	return result, nil
}

func hasResetMarker(group *ast.CommentGroup) bool {
	if group == nil {
		return false
	}
	for _, comment := range group.List {
		text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		if text == resetMarker {
			return true
		}
	}
	return false
}

func renderPackage(generation packageGeneration) ([]byte, error) {
	imports := newImportSet(generation.types, generation.structs)
	var methods bytes.Buffer

	for _, structure := range generation.structs {
		renderMethod(&methods, imports, structure)
	}

	var source bytes.Buffer
	source.WriteString("// Code generated by reset; DO NOT EDIT.\n\n")
	fmt.Fprintf(&source, "package %s\n", generation.name)

	if entries := imports.entries(); len(entries) > 0 {
		source.WriteString("\nimport (\n")
		for _, entry := range entries {
			fmt.Fprintf(&source, "\t%s %q\n", entry.alias, entry.path)
		}
		source.WriteString(")\n")
	}
	source.WriteByte('\n')
	source.Write(methods.Bytes())

	formatted, err := format.Source(source.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated source: %w\n%s", err, source.String())
	}
	return formatted, nil
}

func renderMethod(
	output *bytes.Buffer,
	imports *importSet,
	structure markedStruct,
) {
	names := newNameSet(structure.named)
	receiver := names.unique("r")
	fmt.Fprintf(
		output,
		"func (%s *%s) Reset() {\n",
		receiver,
		receiverType(structure),
	)
	fmt.Fprintf(output, "\tif %s == nil {\n\t\treturn\n\t}\n", receiver)

	fields := structure.named.Underlying().(*types.Struct)
	for i := 0; i < fields.NumFields(); i++ {
		field := fields.Field(i)
		if field.Name() == "_" {
			continue
		}
		output.WriteByte('\n')
		renderReset(output, imports, names, receiver+"."+field.Name(), field.Type(), 1)
	}

	output.WriteString("}\n\n")
}

func receiverType(structure markedStruct) string {
	parameters := structure.named.TypeParams()
	if parameters == nil || parameters.Len() == 0 {
		return structure.name
	}

	names := make([]string, parameters.Len())
	for i := 0; i < parameters.Len(); i++ {
		names[i] = parameters.At(i).Obj().Name()
	}
	return structure.name + "[" + strings.Join(names, ", ") + "]"
}

type nameSet struct {
	used map[string]struct{}
}

func newNameSet(named *types.Named) *nameSet {
	names := &nameSet{used: make(map[string]struct{})}
	if parameters := named.TypeParams(); parameters != nil {
		for i := 0; i < parameters.Len(); i++ {
			names.used[parameters.At(i).Obj().Name()] = struct{}{}
		}
	}
	return names
}

func (names *nameSet) unique(base string) string {
	if _, exists := names.used[base]; !exists {
		names.used[base] = struct{}{}
		return base
	}

	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s%d", base, suffix)
		if _, exists := names.used[candidate]; !exists {
			names.used[candidate] = struct{}{}
			return candidate
		}
	}
}

func renderReset(
	output *bytes.Buffer,
	imports *importSet,
	names *nameSet,
	expression string,
	fieldType types.Type,
	indent int,
) {
	unalias := types.Unalias(fieldType)
	if _, isTypeParameter := unalias.(*types.TypeParam); isTypeParameter {
		renderZero(output, imports, names, expression, fieldType, indent)
		return
	}

	underlying := unalias.Underlying()

	switch typed := underlying.(type) {
	case *types.Basic:
		switch {
		case typed.Info()&types.IsBoolean != 0:
			writeLine(output, indent, expression+" = false")
		case typed.Info()&types.IsString != 0:
			writeLine(output, indent, expression+` = ""`)
		case typed.Info()&types.IsNumeric != 0:
			writeLine(output, indent, expression+" = 0")
		case typed.Kind() == types.UnsafePointer:
			writeLine(output, indent, expression+" = nil")
		default:
			renderZero(output, imports, names, expression, fieldType, indent)
		}

	case *types.Slice:
		writeLine(output, indent, expression+" = "+expression+"[:0]")

	case *types.Map:
		writeLine(output, indent, "clear("+expression+")")

	case *types.Array:
		index := names.unique("_resetIndex")
		writeLine(output, indent, "for "+index+" := range "+expression+" {")
		renderReset(
			output,
			imports,
			names,
			expression+"["+index+"]",
			typed.Elem(),
			indent+1,
		)
		writeLine(output, indent, "}")

	case *types.Pointer:
		writeLine(output, indent, "if "+expression+" != nil {")
		if _, isStruct := types.Unalias(typed.Elem()).Underlying().(*types.Struct); isStruct {
			resetter := names.unique("_resetter")
			ok := names.unique("_resetOK")
			writeLine(
				output,
				indent+1,
				"if "+resetter+", "+ok+" := interface{}("+expression+
					").(interface{ Reset() }); "+ok+" {",
			)
			writeLine(output, indent+2, resetter+".Reset()")
			writeLine(output, indent+1, "} else {")
			renderReset(
				output,
				imports,
				names,
				"(*("+expression+"))",
				typed.Elem(),
				indent+2,
			)
			writeLine(output, indent+1, "}")
		} else {
			renderReset(
				output,
				imports,
				names,
				"(*("+expression+"))",
				typed.Elem(),
				indent+1,
			)
		}
		writeLine(output, indent, "}")

	case *types.Struct:
		resetter := names.unique("_resetter")
		ok := names.unique("_resetOK")
		writeLine(
			output,
			indent,
			"if "+resetter+", "+ok+" := interface{}(&"+expression+
				").(interface{ Reset() }); "+ok+" {",
		)
		writeLine(output, indent+1, resetter+".Reset()")
		writeLine(output, indent, "} else {")
		renderZero(output, imports, names, expression, fieldType, indent+1)
		writeLine(output, indent, "}")

	case *types.Interface, *types.Chan, *types.Signature:
		writeLine(output, indent, expression+" = nil")

	default:
		renderZero(output, imports, names, expression, fieldType, indent)
	}
}

func renderZero(
	output *bytes.Buffer,
	imports *importSet,
	names *nameSet,
	expression string,
	fieldType types.Type,
	indent int,
) {
	zero := names.unique("_resetZero")
	writeLine(output, indent, "{")
	writeLine(
		output,
		indent+1,
		"var "+zero+" "+types.TypeString(fieldType, imports.qualifier),
	)
	writeLine(output, indent+1, expression+" = "+zero)
	writeLine(output, indent, "}")
}

func writeLine(output *bytes.Buffer, indent int, line string) {
	output.WriteString(strings.Repeat("\t", indent))
	output.WriteString(line)
	output.WriteByte('\n')
}

type importEntry struct {
	path  string
	alias string
}

type importSet struct {
	current *types.Package
	byPath  map[string]string
	used    map[string]struct{}
}

func newImportSet(current *types.Package, structs []markedStruct) *importSet {
	imports := &importSet{
		current: current,
		byPath:  make(map[string]string),
		used:    make(map[string]struct{}),
	}
	for _, structure := range structs {
		parameters := structure.named.TypeParams()
		if parameters == nil {
			continue
		}
		for i := 0; i < parameters.Len(); i++ {
			imports.used[parameters.At(i).Obj().Name()] = struct{}{}
		}
	}
	return imports
}

func (imports *importSet) qualifier(pkg *types.Package) string {
	if pkg == nil || pkg.Path() == imports.current.Path() {
		return ""
	}
	if alias, ok := imports.byPath[pkg.Path()]; ok {
		return alias
	}

	index := len(imports.byPath) + 1
	alias := fmt.Sprintf("resetpkg%d", index)
	for {
		if _, exists := imports.used[alias]; !exists {
			break
		}
		index++
		alias = fmt.Sprintf("resetpkg%d", index)
	}
	imports.byPath[pkg.Path()] = alias
	imports.used[alias] = struct{}{}
	return alias
}

func (imports *importSet) entries() []importEntry {
	entries := make([]importEntry, 0, len(imports.byPath))
	for path, alias := range imports.byPath {
		entries = append(entries, importEntry{path: path, alias: alias})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].alias < entries[j].alias
	})
	return entries
}
