package content

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/layer87-labs/webhull/internal/pkg/config"
)

// frontmatter keys that are NOT put into the Content map.
var reservedKeys = map[string]bool{
	"id":              true,
	"template":        true,
	"title":           true,
	"description":     true,
	"keywords":        true,
	"seo_title":       true,
	"seo_description": true,
	"seo_priority":    true,
	"seo_changefreq":  true,
	"seo_ogimage":     true,
	"seo_ogtype":      true,
	"seo_noindex":     true,
	"startPage":       true,
}

// contentFile represents one parsed HTML file with frontmatter.
type contentFile struct {
	Lang     string
	Slug     string // derived from filename
	Meta     map[string]interface{}
	Body     string // raw HTML after frontmatter
	FilePath string
}

// Load scans the content directory for HTML files organized by language,
// parses frontmatter + body, and returns []config.PageConfig ready for
// pages.NewService().
//
// Directory structure:
//
//	contentDir/
//	  de/
//	    start.html     → slug="start", lang="de"
//	    desk.html      → slug="desk", lang="de"
//	  en/
//	    home.html      → slug="home", lang="en"
//	    desk.html      → slug="desk", lang="en"
func Load(contentDir string, languages []string, logger *zap.Logger) ([]config.PageConfig, error) {
	if contentDir == "" {
		return nil, nil
	}

	// Parse all files grouped by language
	var files []contentFile
	for _, lang := range languages {
		langDir := filepath.Join(contentDir, lang)
		langFiles, err := parseDir(langDir, lang)
		if err != nil {
			return nil, fmt.Errorf("language %q: %w", lang, err)
		}
		files = append(files, langFiles...)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no content files found in %s", contentDir)
	}

	// Group by ID → build PageConfig per unique page
	grouped := make(map[string][]contentFile) // id → files
	for _, f := range files {
		id := stringMeta(f.Meta, "id", f.Slug) // default ID = slug
		grouped[id] = append(grouped[id], f)
	}

	// Sort keys for deterministic output
	ids := make([]string, 0, len(grouped))
	for id := range grouped {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	// Build PageConfig slice
	var pages []config.PageConfig
	for _, id := range ids {
		group := grouped[id]

		pageCfg := config.PageConfig{
			ID:   id,
			I18n: make(map[string]config.PageI18nConfig),
		}

		for _, f := range group {
			// Template: first file with a template wins (usually all have the same)
			if tmpl := stringMeta(f.Meta, "template", ""); tmpl != "" && pageCfg.Template == "" {
				pageCfg.Template = tmpl
			}

			// SEO (from any language file — they share the same PageSEOConfig)
			if v := floatMeta(f.Meta, "seo_priority"); v > 0 {
				pageCfg.SEO.Priority = v
			}
			if v := stringMeta(f.Meta, "seo_changefreq", ""); v != "" {
				pageCfg.SEO.ChangeFreq = v
			}
			if v := stringMeta(f.Meta, "seo_ogimage", ""); v != "" {
				pageCfg.SEO.OGImage = v
			}
			if v := stringMeta(f.Meta, "seo_ogtype", ""); v != "" {
				pageCfg.SEO.OGType = v
			}
			if boolMeta(f.Meta, "seo_noindex") {
				pageCfg.SEO.NoIndex = true
			}

			// Build content map from non-reserved frontmatter keys
			content := make(map[string]string)
			for key, val := range f.Meta {
				if reservedKeys[key] {
					continue
				}
				content[key] = fmt.Sprintf("%v", val)
			}

			// Parse the body: typed section markers produce an ordered Sections list;
			// named markers (legacy) or plain body produce the flat Content map.
			var sectionList []config.SectionConfig
			if f.Body != "" {
				if hasSectionList := strings.Contains(f.Body, typedSectionMarkerPrefix); hasSectionList {
					sectionList = parseSectionList(f.Body)
				} else {
					sections := parseSections(f.Body)
					for key, val := range sections {
						content[key] = val
					}
				}
			}

			pageCfg.I18n[f.Lang] = config.PageI18nConfig{
				Slug:        f.Slug,
				Title:       stringMeta(f.Meta, "title", f.Slug),
				SEOTitle:    stringMeta(f.Meta, "seo_title", ""),
				Description: stringMeta(f.Meta, "description", ""),
				SEODesc:     stringMeta(f.Meta, "seo_description", ""),
				Keywords:    stringMeta(f.Meta, "keywords", ""),
				Content:     content,
				Sections:    sectionList,
			}
		}

		// Default template
		if pageCfg.Template == "" {
			pageCfg.Template = "default"
		}

		pages = append(pages, pageCfg)

		logger.Debug("content page loaded",
			zap.String("id", id),
			zap.String("template", pageCfg.Template),
			zap.Int("languages", len(pageCfg.I18n)))
	}

	logger.Info("content loaded",
		zap.String("dir", contentDir),
		zap.Int("pages", len(pages)),
		zap.Int("files", len(files)))

	return pages, nil
}

// parseDir reads all .html files from a language directory.
func parseDir(dir, lang string) ([]contentFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var files []contentFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".html") {
			continue
		}

		path := filepath.Join(dir, name)
		cf, err := parseFile(path, lang)
		if err != nil {
			return nil, fmt.Errorf("file %s: %w", name, err)
		}
		files = append(files, cf)
	}

	return files, nil
}

// parseFile reads an HTML file with optional YAML frontmatter.
// Frontmatter is delimited by --- on its own line.
func parseFile(path, lang string) (contentFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return contentFile{}, fmt.Errorf("failed to open: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	// Derive slug from filename
	slug := strings.TrimSuffix(filepath.Base(path), ".html")

	cf := contentFile{
		Lang:     lang,
		Slug:     slug,
		Meta:     make(map[string]interface{}),
		FilePath: path,
	}

	// Check for frontmatter (first line must be "---")
	if !scanner.Scan() {
		return cf, nil // empty file
	}

	firstLine := strings.TrimSpace(scanner.Text())
	if firstLine != "---" {
		// No frontmatter — entire file is body
		var body strings.Builder
		body.WriteString(scanner.Text())
		body.WriteString("\n")
		for scanner.Scan() {
			body.WriteString(scanner.Text())
			body.WriteString("\n")
		}
		cf.Body = strings.TrimSpace(body.String())
		return cf, nil
	}

	// Read frontmatter until closing ---
	var fmBuilder strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		fmBuilder.WriteString(line)
		fmBuilder.WriteString("\n")
	}

	// Parse frontmatter as YAML
	if fmBuilder.Len() > 0 {
		if err := yaml.Unmarshal([]byte(fmBuilder.String()), &cf.Meta); err != nil {
			return contentFile{}, fmt.Errorf("invalid frontmatter YAML: %w", err)
		}
	}

	// Rest is body HTML
	var bodyBuilder strings.Builder
	for scanner.Scan() {
		bodyBuilder.WriteString(scanner.Text())
		bodyBuilder.WriteString("\n")
	}
	cf.Body = strings.TrimSpace(bodyBuilder.String())

	return cf, nil
}

// Helper functions to extract typed values from frontmatter map.

func stringMeta(meta map[string]interface{}, key, fallback string) string {
	if v, ok := meta[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return fallback
}

func boolMeta(meta map[string]interface{}, key string) bool {
	if v, ok := meta[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func floatMeta(meta map[string]interface{}, key string) float64 {
	if v, ok := meta[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		}
	}
	return 0
}

// sectionMarkerPrefix is the HTML comment prefix for named content sections.
const sectionMarkerPrefix = "<!-- section: "

// typedSectionMarkerPrefix is the HTML comment prefix for typed ordered sections.
const typedSectionMarkerPrefix = "<!-- section["

// parseSections splits the body HTML into named content sections.
// If the body contains <!-- section: name --> markers, each marker starts a
// new named section. Content before the first marker goes into "body".
// If no markers are found, the entire body goes into "body".
func parseSections(body string) map[string]string {
	sections := make(map[string]string)

	// Fast path: no section markers at all
	if !strings.Contains(body, sectionMarkerPrefix) {
		sections["body"] = body
		return sections
	}

	currentKey := "body"
	var current strings.Builder
	lines := strings.Split(body, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, sectionMarkerPrefix) && strings.HasSuffix(trimmed, "-->") {
			// Save previous section
			if content := strings.TrimSpace(current.String()); content != "" {
				sections[currentKey] = content
			}
			// Extract new section name
			name := trimmed[len(sectionMarkerPrefix) : len(trimmed)-len("-->")]
			currentKey = strings.TrimSpace(name)
			current.Reset()
		} else {
			current.WriteString(line)
			current.WriteString("\n")
		}
	}

	// Save last section
	if content := strings.TrimSpace(current.String()); content != "" {
		sections[currentKey] = content
	}

	return sections
}

// parseSectionList parses a body containing typed section markers into an ordered
// list of SectionConfig. The marker format is:
//
//	<!-- section[type]: Title HTML -->
//	<!-- section[type,altbg]: Title HTML -->
//	<!-- section[type,altbg,id=anchor]: Title HTML -->
//
// Supported types: "block" (content-block), "grid" (content-grid), "services" (services-grid).
// The altbg flag adds the alt-bg CSS modifier. id sets the HTML id on the section element.
// Title is raw HTML rendered in the <h2> header; an empty title omits the header entirely.
// Content following the marker until the next marker (or end of body) is the section body.
func parseSectionList(body string) []config.SectionConfig {
	var sections []config.SectionConfig
	var current *config.SectionConfig
	var currentBody strings.Builder

	lines := strings.Split(body, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, typedSectionMarkerPrefix) && strings.HasSuffix(trimmed, "-->") {
			// Save previous section body before starting a new one.
			if current != nil {
				current.Body = strings.TrimSpace(currentBody.String())
				sections = append(sections, *current)
				currentBody.Reset()
			}

			// Parse: <!-- section[attrs]: Title -->
			// Strip prefix and suffix, leaving: "attrs]: Title "
			inner := trimmed[len(typedSectionMarkerPrefix):]
			bracketEnd := strings.Index(inner, "]:")
			if bracketEnd < 0 {
				// Malformed marker — skip.
				continue
			}

			rawAttrs := inner[:bracketEnd]
			rawTitle := strings.TrimSuffix(strings.TrimSpace(inner[bracketEnd+2:]), "-->")
			rawTitle = strings.TrimSpace(rawTitle)

			entry := config.SectionConfig{Title: rawTitle}
			for i, attr := range strings.Split(rawAttrs, ",") {
				attr = strings.TrimSpace(attr)
				switch {
				case i == 0:
					entry.Type = attr
				case attr == "altbg":
					entry.AltBg = true
				case strings.HasPrefix(attr, "id="):
					entry.ID = strings.TrimPrefix(attr, "id=")
				}
			}
			current = &entry
		} else if current != nil {
			currentBody.WriteString(line)
			currentBody.WriteString("\n")
		}
		// Lines before the first marker are ignored.
	}

	// Save the last section.
	if current != nil {
		current.Body = strings.TrimSpace(currentBody.String())
		sections = append(sections, *current)
	}

	return sections
}
