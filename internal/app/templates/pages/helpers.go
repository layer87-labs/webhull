package pages

import "github.com/a-h/templ"

// sectionClass returns the CSS class for the outer <section> element.
func sectionClass(sType string, altBg bool) string {
	base := "content-section"
	if sType == "services" {
		base = "services"
	}
	if altBg {
		return base + " alt-bg"
	}
	return base
}

// sectionInnerClass returns the CSS class for the inner content wrapper.
func sectionInnerClass(sType string) string {
	switch sType {
	case "grid":
		return "content-grid"
	case "services":
		return "services-grid"
	default: // "block" and any unknown type
		return "content-block"
	}
}

// sectionIDAttr returns a templ.Attributes map with an "id" entry when id is
// non-empty, or an empty map otherwise, enabling conditional id attributes.
func sectionIDAttr(id string) templ.Attributes {
	if id == "" {
		return templ.Attributes{}
	}
	return templ.Attributes{"id": id}
}
