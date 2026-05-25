package navigation

import (
	"github.com/layer87-labs/webhull/internal/pkg/config"
	"github.com/layer87-labs/webhull/internal/pkg/i18n"
)

// Service resolves navigation items for the current language and active page.
type Service struct {
	header map[i18n.Language][]config.NavItemConfig
	footer map[i18n.Language][]config.NavItemConfig
}

// NewService creates a navigation service from configuration.
func NewService(navCfg config.NavigationConfig) *Service {
	header := make(map[i18n.Language][]config.NavItemConfig)
	for lang, items := range navCfg.Header {
		header[i18n.Language(lang)] = items
	}

	footer := make(map[i18n.Language][]config.NavItemConfig)
	for lang, items := range navCfg.Footer {
		footer[i18n.Language(lang)] = items
	}

	return &Service{
		header: header,
		footer: footer,
	}
}

// ResolveHeader returns header navigation items for the given language,
// with the active state set based on the current slug.
func (s *Service) ResolveHeader(lang i18n.Language, currentSlug string) Header {
	items := s.resolveItems(s.header[lang], currentSlug)
	return Header{
		Items:    items,
		Language: lang,
	}
}

// ResolveFooter returns footer navigation items for the given language.
func (s *Service) ResolveFooter(lang i18n.Language, currentSlug string) Footer {
	items := s.resolveItems(s.footer[lang], currentSlug)
	return Footer{
		Items:    items,
		Language: lang,
	}
}

// resolveItems converts config items to navigation items with active state.
func (s *Service) resolveItems(cfgItems []config.NavItemConfig, currentSlug string) []Item {
	items := make([]Item, len(cfgItems))
	for idx, cfg := range cfgItems {
		item := Item{
			Title:  cfg.Title,
			URL:    cfg.URL,
			Active: isActive(cfg, currentSlug),
		}

		if len(cfg.Children) > 0 {
			item.Children = s.resolveItems(cfg.Children, currentSlug)
			// Parent is active if any child is active
			for _, child := range item.Children {
				if child.Active {
					item.Active = true
					break
				}
			}
		}

		items[idx] = item
	}
	return items
}

// isActive checks if a nav item matches the current slug.
func isActive(cfg config.NavItemConfig, currentSlug string) bool {
	if cfg.Slug != "" {
		return cfg.Slug == currentSlug
	}
	// Fallback: compare URL path (strip leading /)
	url := cfg.URL
	if len(url) > 0 && url[0] == '/' {
		url = url[1:]
	}
	return url == currentSlug
}
