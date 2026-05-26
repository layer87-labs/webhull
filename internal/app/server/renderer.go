package server

import (
	"bytes"
	"context"
	"fmt"

	"github.com/a-h/templ"

	"github.com/layer87-labs/webhull/internal/app/templates"
	tmplpages "github.com/layer87-labs/webhull/internal/app/templates/pages"
)

// templateRegistry maps template names to their templ components.
var templateRegistry = map[string]func(*templates.PageData) templ.Component{
	"home":     func(d *templates.PageData) templ.Component { return tmplpages.HomePage(d) },
	"contact":  func(d *templates.PageData) templ.Component { return tmplpages.ContactPage(d) },
	"legal":    func(d *templates.PageData) templ.Component { return tmplpages.LegalPage(d) },
	"default":  func(d *templates.PageData) templ.Component { return tmplpages.DefaultPage(d) },
	"single":   func(d *templates.PageData) templ.Component { return tmplpages.SinglePage(d) },
	"notfound": func(d *templates.PageData) templ.Component { return tmplpages.NotFoundPage(d) },
}

// renderTemplate renders a page template to bytes.
func renderTemplate(ctx context.Context, templateName string, data *templates.PageData) ([]byte, error) {
	fn, ok := templateRegistry[templateName]
	if !ok {
		fn = templateRegistry["default"]
	}

	component := fn(data)

	var buf bytes.Buffer
	if err := component.Render(ctx, &buf); err != nil {
		return nil, fmt.Errorf("render template %q: %w", templateName, err)
	}

	return buf.Bytes(), nil
}
