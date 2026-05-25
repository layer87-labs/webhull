---
description: Add a new templ template component to webhull
argument-hint: "<component-name> [page|layout]"
---
Add a new templ template component `$1` to webhull.

Type: `$2` (default: layout if omitted — goes in `internal/app/templates/layout/`; use `page` for `internal/app/templates/pages/`)

Steps:
1. Create `internal/app/templates/$2/$1.templ` with the component function signature
2. Run `make generate` to produce the companion `_templ.go` file
3. Commit BOTH `$1.templ` AND `$1_templ.go` — the `_templ.go` is committed to git, CI does not generate it

The view model passed to page templates is `PageData` (see `internal/app/templates/viewmodels.go`).
Use `data.AssetPath("/static/...")` for all static asset references — never bare paths.
Use `data.Content("key")` / `data.HasContent("key")` to access named content sections.

Do NOT add business logic inside `.templ` files — keep them pure presentation.
