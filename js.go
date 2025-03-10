// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package flyscrape

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/evanw/esbuild/pkg/api"
)

//go:embed template.js
var ScriptTemplate []byte

type Config []byte

type ScrapeParams struct {
	HTML    string
	URL     string
	Process func(url string) ([]byte, error)
	Follow  func(url string)
}

type ScrapeFunc func(ScrapeParams) (any, error)

type TransformError struct {
	Line   int
	Column int
	Text   string
}

func (err TransformError) Error() string {
	return fmt.Sprintf("%d:%d: %s", err.Line, err.Column, err.Text)
}

type Exports map[string]any

func (e Exports) Config() []byte {
	b, _ := json.Marshal(e["config"])
	return b
}

func (e Exports) Scrape(p ScrapeParams) (any, error) {
	fn := e["__scrape"].(ScrapeFunc)
	return fn(p)
}

type Imports map[string]map[string]any

func Compile(src string, imports Imports) (Exports, error) {
	src, err := build(src)
	if err != nil {
		return nil, err
	}
	return vm(src, imports)
}

func build(src string) (string, error) {
	res := api.Build(api.BuildOptions{
		Loader: map[string]api.Loader{
			".txt":  api.LoaderText,
			".json": api.LoaderJSON,
		},
		Bundle: true,
		Stdin: &api.StdinOptions{
			Contents:   src,
			ResolveDir: ".",
		},
		Platform: api.PlatformNode,
		Format:   api.FormatCommonJS,
		External: []string{"flyscrape"},
	})

	var errs []error
	for _, msg := range res.Errors {
		err := TransformError{Text: msg.Text}
		if msg.Location != nil {
			err.Line = msg.Location.Line
			err.Column = msg.Location.Column
		}
		errs = append(errs, err)
	}
	if len(res.Errors) > 0 {
		return "", errors.Join(errs...)
	}
	if len(res.OutputFiles) == 0 {
		return "", errors.New("no output generated")
	}

	return string(res.OutputFiles[0].Contents), nil
}

func vm(src string, imports Imports) (Exports, error) {
	vm := goja.New()
	registry := &require.Registry{}

	registry.Enable(vm)
	console.Enable(vm)

	for module, pkg := range imports {
		pkg := pkg
		registry.RegisterNativeModule(module, func(vm *goja.Runtime, o *goja.Object) {
			exports := vm.NewObject()

			for ident, val := range pkg {
				exports.Set(ident, val)
			}

			o.Set("exports", exports)
		})
	}

	if _, err := vm.RunString("module = {}"); err != nil {
		return nil, fmt.Errorf("running defining module: %w", err)
	}
	if _, err := vm.RunString(src); err != nil {
		return nil, fmt.Errorf("running user script: %w", err)
	}

	v, err := vm.RunString("module.exports")
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	exports := Exports{}
	if goja.IsUndefined(v) {
		return exports, nil
	}

	obj := v.ToObject(vm)
	for _, key := range obj.Keys() {
		exports[key] = obj.Get(key).Export()
	}

	exports["__scrape"], err = scrape(vm)
	if err != nil {
		return nil, err
	}

	return exports, nil
}

func scrape(vm *goja.Runtime) (ScrapeFunc, error) {
	var lock sync.Mutex

	if v, err := vm.RunString("module.exports.default"); err != nil || goja.IsUndefined(v) {
		return nil, errors.New("default export is not defined")
	}

	defaultfn, err := vm.RunString("(o) => JSON.stringify(module.exports.default(o))")
	if err != nil {
		return nil, fmt.Errorf("failed to create scrape function: %w", err)
	}

	scrapefn, ok := defaultfn.Export().(func(goja.FunctionCall) goja.Value)
	if !ok {
		return nil, errors.New("failed to export scrape function")
	}

	var newArg func(p ScrapeParams) (*goja.Object, error)
	newArg = func(p ScrapeParams) (*goja.Object, error) {
		doc, err := DocumentFromString(p.HTML)
		if err != nil {
			return nil, err
		}

		baseurl, err := url.Parse(p.URL)
		if err != nil {
			return nil, err
		}

		absoluteURL := func(ref string) string {
			abs, err := baseurl.Parse(ref)
			if err != nil {
				return ref
			}
			return abs.String()
		}

		o := vm.NewObject()
		o.Set("url", p.URL)
		o.Set("doc", doc)
		o.Set("absoluteURL", absoluteURL)
		o.Set("scrape", func(url string, f func(goja.FunctionCall) goja.Value) goja.Value {
			url = absoluteURL(url)

			html, err := p.Process(url)
			if err != nil {
				return vm.ToValue(map[string]any{"error": err.Error()})
			}

			newp := ScrapeParams{
				HTML:    string(html),
				URL:     url,
				Process: p.Process,
			}

			arg, err := newArg(newp)
			if err != nil {
				return vm.ToValue(map[string]any{"error": err.Error()})
			}

			return f(goja.FunctionCall{Arguments: []goja.Value{arg}})
		})
		o.Set("follow", func(url string) {
			p.Follow(absoluteURL(url))
		})

		return o, nil
	}

	return func(p ScrapeParams) (any, error) {
		lock.Lock()
		defer lock.Unlock()

		arg, err := newArg(p)
		if err != nil {
			return nil, err
		}

		ret := scrapefn(goja.FunctionCall{Arguments: []goja.Value{arg}})
		if goja.IsUndefined(ret) {
			return nil, nil
		}

		var result any
		if err := json.Unmarshal([]byte(ret.String()), &result); err != nil {
			log.Println(err)
			return nil, err
		}

		return result, nil
	}, nil
}

func DocumentFromString(s string) (map[string]any, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(s))
	if err != nil {
		return nil, err
	}

	return Document(doc.Selection), nil
}

func Document(sel *goquery.Selection) map[string]any {
	o := map[string]any{}
	o["WARNING"] = "Forgot to call text(), html() or attr()?"
	o["text"] = sel.Text
	o["name"] = func() string { return sel.Get(0).Data }
	o["html"] = func() string { h, _ := goquery.OuterHtml(sel); return h }
	o["attr"] = func(name string) string { v, _ := sel.Attr(name); return v }
	o["hasAttr"] = func(name string) bool { _, ok := sel.Attr(name); return ok }
	o["hasClass"] = sel.HasClass
	o["length"] = sel.Length()
	o["first"] = func() map[string]any { return Document(sel.First()) }
	o["last"] = func() map[string]any { return Document(sel.Last()) }
	o["get"] = func(index int) map[string]any { return Document(sel.Eq(index)) }
	o["find"] = func(s string) map[string]any { return Document(sel.Find(s)) }
	o["next"] = func() map[string]any { return Document(sel.Next()) }
	o["nextAll"] = func() map[string]any { return Document(sel.NextAll()) }
	o["nextUntil"] = func(s string) map[string]any { return Document(sel.NextUntil(s)) }
	o["prev"] = func() map[string]any { return Document(sel.Prev()) }
	o["prevAll"] = func() map[string]any { return Document(sel.PrevAll()) }
	o["prevUntil"] = func(s string) map[string]any { return Document(sel.PrevUntil(s)) }
	o["siblings"] = func() map[string]any { return Document(sel.Siblings()) }
	o["children"] = func() map[string]any { return Document(sel.Children()) }
	o["parent"] = func() map[string]any { return Document(sel.Parent()) }
	o["map"] = func(callback func(map[string]any, int) any) []any {
		var vals []any
		sel.Map(func(i int, s *goquery.Selection) string {
			vals = append(vals, callback(Document(s), i))
			return ""
		})
		return vals
	}
	o["filter"] = func(callback func(map[string]any, int) bool) []any {
		var vals []any
		sel.Each(func(i int, s *goquery.Selection) {
			el := Document(s)
			ok := callback(el, i)
			if ok {
				vals = append(vals, el)
			}
		})
		return vals
	}
	return o
}
