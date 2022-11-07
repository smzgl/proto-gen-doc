package build

import (
	"embed"
	htmlTemplate "html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Masterminds/sprig"
)

//go:embed tmpl
var templateFS embed.FS

// Renderer TODO
type Renderer struct {
	tmpl     *Template
	ErrFile  *File
	Packages []*Package
}

// Package TODO
type Package struct {
	Name     string
	Services []*Service
}

// Render TODO
func (r *Renderer) Render(path string) error {
	var err error

	packages := make(map[string]*Package)

	for _, file := range r.tmpl.Files {
		pkg, ok := packages[file.Package]
		if !ok {
			pkg = &Package{
				Name: file.Package,
			}

			packages[pkg.Name] = pkg
		}

		pkg.Services = append(pkg.Services, file.Services...)
	}

	for _, pkg := range packages {
		if strings.HasSuffix(pkg.Name, ".ErrCode") {
			continue
		}

		sort.Slice(pkg.Services, func(i, j int) bool {
			return strings.Compare(pkg.Services[i].FullName, pkg.Services[j].FullName) < 0
		})

		r.Packages = append(r.Packages, pkg)
	}

	sort.Slice(r.Packages, func(i, j int) bool {
		return strings.Compare(r.Packages[i].Name, r.Packages[j].Name) < 0
	})

	for _, file := range r.tmpl.Files {
		if strings.HasSuffix(file.Package, ".ErrCode") {
			r.ErrFile = file
		}
	}

	err = r.renderTOC(path, "tmpl/proto.toc.md.tmpl", "proto.md")
	if err != nil {
		return err
	}

	err = r.renderService(path, "tmpl/proto.doc.md.tmpl", "proto.md")
	if err != nil {
		return err
	}

	return nil
}

func (r *Renderer) createFile(filename string) (*os.File, error) {
	filename = filepath.Clean(filename)

	err := os.MkdirAll(filepath.Dir(filename), 0755)
	if err != nil {
		return nil, err
	}

	return os.Create(filename)
}

func (r *Renderer) renderTOC(path, templateFile, outputFile string) error {
	var err error

	templateText, err := templateFS.ReadFile(templateFile)
	if err != nil {
		return err
	}

	template := htmlTemplate.New("TOC Template").Funcs(funcMap).Funcs(sprig.HtmlFuncMap())
	_, err = template.Parse(string(templateText))
	if err != nil {
		return err
	}

	fp, err := r.createFile(filepath.Join(path, outputFile))
	if err != nil {
		return err
	}

	err = template.Execute(fp, r)
	_ = fp.Close()

	if err != nil {
		return err
	}

	return nil
}

func (r *Renderer) renderService(path, templateFile, outputFile string) error {
	var err error

	templateText, err := templateFS.ReadFile(templateFile)
	if err != nil {
		return err
	}

	template := htmlTemplate.New("Service Template").Funcs(funcMap).Funcs(sprig.HtmlFuncMap())
	_, err = template.Parse(string(templateText))
	if err != nil {
		return err
	}

	for _, file := range r.tmpl.Files {
		fp, err := r.createFile(filepath.Join(path, file.Dir, outputFile))
		if err != nil {
			return err
		}

		err = template.Execute(fp, file)
		_ = fp.Close()

		if err != nil {
			return err
		}
	}

	return nil
}
