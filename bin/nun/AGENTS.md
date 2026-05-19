# nun Agent Notes

`nun` is allowed to change this Nix config, so every command that writes files must use the same confirmation flow:

1. Collect all required inputs first.
2. Print a short summary before writing anything.
3. List every file that will be added, modified, or deleted.
4. Ask for explicit `y` confirmation.
5. Abort without writing files for any other answer.

Keep file-generation logic small and shared. Prefer common helpers for prompts, summaries, confirmation, and file writes instead of one-off command code.

## Architecture Overview

### Command Structure
Commands are defined in `cmd/nun/main.go`. Each command follows this pattern:
- Parse arguments/flags
- Collect data and create a plan (PlanXXX functions)
- Show confirmation UI using `ui.ShowPlan()`
- Apply the plan if confirmed (ApplyXXX functions)

### Package Management

Packages are stored in several locations:

1. **package-sets/*.nix** - Global package sets (development.nix, global.nix, darwin.nix)
   - Simple list format: `pkgs: with pkgs; [ pkg1 pkg2 ]`
   - Darwin.nix has named lists: `homebrewBrews`, `homebrewCasks`, `nixPackages`

2. **hosts/<hostname>/packages.nix** - Host-specific packages
   - Usually imports from package-sets
   - Can contain inline package lists using `home.packages`

3. **hosts/<hostname>/homebrew.nix** - Homebrew packages per host
   - Contains `brews` and `casks` named lists
   - Example: `{ homebrew = { casks = [ "app1" ]; brews = [ "pkg1" ]; }; }`

### Nix Expression Parsing (nixexpr/)

The `internal/nixexpr` package provides primitives for manipulating Nix files:

- **ParseTopLevelList**: Extracts packages from simple lists like `with pkgs; [ ... ]`
- **ParseNamedList**: Extracts packages from named lists like `brews = [ ... ]`
- **AddListItem/AddNamedListItem**: Add items to lists while preserving formatting
- **RemoveListItem/RemoveNamedListItem**: Remove items from lists
- **AddListItemAfter**: Add after a specific marker (used for host packages)

Key insight: Nix parsing is line-based and regex-driven, not a full AST parser. This works because:
- Package lists are simple and follow consistent patterns
- Comments are stripped before parsing
- Indentation is preserved for clean output

### UI Components (ui/)

Bubble Tea (TUI framework) is used for interactive interfaces:

- **ShowPlan**: Confirmation dialog with Apply/Cancel/Interactive options
  - Handles 'y' for apply, 'n' for cancel, 'i' for interactive
  - Tab/h/l to move between actions

- **SelectPackagesToRemove**: Multi-select list with space to toggle
  - Shows packages from all sources (nix + brew)
  - / to filter, enter to confirm, esc/q to cancel
  - Displays [x] for selected, [ ] for unselected

- **PickHost**: Single-select from a list of hosts
- **NewHost**: Multi-step form for creating new hosts
- **BrowsePackages**: Read-only package browser

### Configuration (config/)

The `App` struct is the main entry point for all config operations:

- **Repo root discovery**: Uses git, env var, or ~/nix-config fallback
- **Package operations**: Install, Remove, Try (temporary install)
- **Host operations**: Create new hosts, list hosts, print current host
- **Dotfile operations**: Link, ingest (move file to dotfiles)

Key types:
- `InstallTarget`: What to install (kind, package, package set)
- `RemoveTarget`: What to remove (includes host/global info for brew)
- `PendingWrite`: File change to be applied (path, relative path, content)

### Adding New Commands

1. Add command to help text and switch statement in main.go
2. Create runXXX function following the confirmation flow
3. Add PlanXXX and ApplyXXX methods to App in config/
4. Add UI components to ui/ if needed
5. Add nixexpr primitives if manipulating Nix files
6. Update tests

### Common Patterns

**Interactive mode (-i flag):**
```go
if interactive {
    result, err := ui.SelectXXX(data)
    if result.Aborted { return nil }
    if result.Cancelled { return nil }
    // Convert result to request
}
```

**Building a modification plan:**
```go
files := map[string]string{}
for _, target := range targets {
    path, _, _ := a.location(root, target)
    source := files[path] // or read from disk
    next := modifySource(source, target)
    files[path] = next
}
// Convert files map to []PendingWrite
```

**Confirmation flow:**
```go
action, err := ui.ShowPlan(ui.PlanView{
    Title:   "nun command",
    Summary: "Description of what will happen.",
    Sections: []ui.PlanSection{
        {Title: "Section name", Items: items},
        {Title: "Files to modify", Items: filePaths},
    },
    Actions: []ui.PlanAction{ui.PlanApply, ui.PlanCancel},
})
if action != ui.PlanApply {
    fmt.Println("aborted")
    return nil
}
return app.ApplyXXX(plan, os.Stdout)
```

### File Manipulation Safety

- Always use `PendingWrite` to stage changes before applying
- `ApplyWrites` creates directories with 0755, files with 0644
- Homebrew tap wiring is automatic when adding packages with `/` in name
- Backups are created with timestamp suffix for dotfile operations

### Testing

- Unit tests in `*_test.go` files alongside source
- Use testdata/ directories for complex fixtures if needed
- Test nixexpr functions with real Nix syntax examples
- UI components are harder to test; focus on model logic

### Gotchas

- Nix list items may or may not have trailing commas
- Homebrew packages in lists are quoted strings (e.g., `"docker-desktop"`)
- Nix packages are unquoted identifiers (e.g., `jq`, `ripgrep`)
- Package sets and hosts are separate namespaces
- Darwin.nix uses different list names than homebrew.nix
