package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"nun/internal/config"
	"nun/internal/ui"
)

const help = `nun config helper

Usage:
  nun rebuild [host] [--remote] [nh flags...] [-- nix flags...]
  nun packages
  nun hosts
  nun hosts new
  nun try [--brew|--cask] <package>...
  nun try --profile [host]
  nun install [-i] [--set package-set] [--brew|--cask] [package...]
  nun link
  nun ingest <file>
  nun remove [-i] [package...]

Commands:
  rebuild   Rebuild this nix-darwin/NixOS config with nh
  packages  Browse package-sets/*.nix in a terminal view
  hosts     Print the current host, or create a new host with 'nun hosts new'
  try       Temporarily install packages, or dry-check a host profile
  install   Temporarily install packages and add them to package lists
  link      Symlink repo-managed dotfiles into this user account
  ingest    Move a file to dotfiles, add to links.nix, and create symlink
  remove    Remove packages from package lists (does not uninstall)
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Print(help)
		return nil
	}

	app := config.NewApp()
	switch args[0] {
	case "rebuild":
		return app.Rebuild(args[1:])
	case "packages":
		entries, err := app.ReadPackageSets()
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			return errors.New("no packages found in package-sets/*.nix")
		}
		return ui.BrowsePackages(entries)
	case "hosts":
		return runHosts(app, args[1:])
	case "try":
		return runTry(app, args[1:])
	case "install":
		return runInstall(app, args[1:])
	case "link":
		return runLink(app, args[1:])
	case "ingest":
		return runIngest(app, args[1:])
	case "remove":
		return runRemove(app, args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runLink(app config.App, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: nun link")
	}
	links, err := app.PlanDotfileLinks()
	if err != nil {
		return err
	}
	action, err := ui.ShowPlan(ui.PlanView{
		Title:   "nun link",
		Summary: "Symlink repo-managed dotfiles into this user account.",
		Sections: []ui.PlanSection{
			{Title: "Links", Items: config.DotfilePlanItems(links)},
		},
		Actions: []ui.PlanAction{ui.PlanApply, ui.PlanCancel},
	})
	if err != nil {
		return err
	}
	if action != ui.PlanApply {
		fmt.Println("aborted")
		return nil
	}
	return app.ApplyDotfileLinks(links, os.Stdout)
}

func runIngest(app config.App, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: nun ingest <file>")
	}
	filePath := args[0]
	
	result, err := app.PlanIngest(filePath)
	if err != nil {
		return err
	}
	
	action, err := ui.ShowPlan(ui.PlanView{
		Title:   "nun ingest",
		Summary: "Move a file to dotfiles and create a symlink.",
		Sections: []ui.PlanSection{
			{Title: "Source", Items: []string{result.SourcePath}},
			{Title: "Destination", Items: []string{result.DotfilesPath}},
			{Title: "Symlink", Items: []string{result.TargetPath + " -> " + result.DotfilesPath}},
			{Title: "Files to modify", Items: []string{"links.nix"}},
		},
		Actions: []ui.PlanAction{ui.PlanApply, ui.PlanCancel},
	})
	if err != nil {
		return err
	}
	if action != ui.PlanApply {
		fmt.Println("aborted")
		return nil
	}
	return app.ApplyIngest(result, os.Stdout)
}

func runRemove(app config.App, args []string) error {
	interactive := false
	var packages []string
	
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-i", "--interactive":
			interactive = true
		case "-h", "--help":
			fmt.Print(`usage: nun remove [-i] [package...]

Remove packages from package lists. This only edits the nix files;
it does not uninstall already-installed packages.

Flags:
  -i, --interactive  Select packages interactively from a list
  -h, --help        Show this help message

Examples:
  nun remove jq                    # Remove jq from its package set
  nun remove -i                    # Interactive mode: select from list
`)
			return nil
		default:
			if strings.HasPrefix(args[i], "-") {
				return fmt.Errorf("unknown flag: %s", args[i])
			}
			packages = append(packages, args[i])
		}
	}
	
	if interactive {
		return runRemoveInteractive(app)
	}
	
	if len(packages) == 0 {
		return fmt.Errorf("usage: nun remove [-i] [package...]")
	}
	
	req := config.RemoveRequest{
		Packages: packages,
	}
	
	plan, err := app.PlanRemove(req)
	if err != nil {
		return err
	}
	
	action, err := ui.ShowPlan(ui.PlanView{
		Title:   "nun remove",
		Summary: "Remove packages from package lists.",
		Sections: []ui.PlanSection{
			{Title: "Packages to remove", Items: describeRemoveTargets(plan.Targets)},
			{Title: "Files to modify", Items: describeWrites(plan.Writes)},
		},
		Actions: []ui.PlanAction{ui.PlanApply, ui.PlanCancel},
	})
	if err != nil {
		return err
	}
	if action != ui.PlanApply {
		fmt.Println("aborted")
		return nil
	}
	return app.ApplyRemove(plan, os.Stdout)
}

func runRemoveInteractive(app config.App) error {
	nixPackages, brewPackages, err := app.AllPackages()
	if err != nil {
		return err
	}
	if len(nixPackages) == 0 && len(brewPackages) == 0 {
		return fmt.Errorf("no packages found in any package set or homebrew config")
	}

	result, err := ui.SelectPackagesToRemove(nixPackages, brewPackages)
	if err != nil {
		return err
	}
	if result.Aborted {
		fmt.Println("aborted")
		return nil
	}
	if result.Cancelled {
		return nil
	}
	if len(result.Selected) == 0 {
		fmt.Println("no packages selected")
		return nil
	}

	// Convert to remove request
	var packages []string
	for _, entry := range result.Selected {
		if entry.Kind == "nix" {
			packages = append(packages, entry.Set+"/"+entry.Name)
		} else if entry.IsGlobal {
			packages = append(packages, "darwin/"+entry.Name)
		} else if entry.Host != "" {
			packages = append(packages, entry.Host+"/"+entry.Name)
		} else {
			packages = append(packages, entry.Name)
		}
	}

	req := config.RemoveRequest{
		Packages:    packages,
		Interactive: true,
	}

	plan, err := app.PlanRemove(req)
	if err != nil {
		return err
	}

	action, err := ui.ShowPlan(ui.PlanView{
		Title:   "nun remove",
		Summary: "Remove selected packages from package lists.",
		Sections: []ui.PlanSection{
			{Title: "Packages to remove", Items: describeRemoveTargets(plan.Targets)},
			{Title: "Files to modify", Items: describeWrites(plan.Writes)},
		},
		Actions: []ui.PlanAction{ui.PlanApply, ui.PlanCancel},
	})
	if err != nil {
		return err
	}
	if action != ui.PlanApply {
		fmt.Println("aborted")
		return nil
	}
	return app.ApplyRemove(plan, os.Stdout)
}

func describeRemoveTargets(targets []config.RemoveTarget) []string {
	items := make([]string, len(targets))
	for i, target := range targets {
		switch target.Kind {
		case config.InstallNix:
			items[i] = fmt.Sprintf("%s from %s", target.Package, target.PackageSet)
		case config.InstallBrew, config.InstallCask:
			location := target.Host
			if target.IsGlobal {
				location = "darwin"
			}
			items[i] = fmt.Sprintf("%s %s from %s", target.Kind, target.Package, location)
		default:
			items[i] = fmt.Sprintf("%s from %s", target.Package, target.PackageSet)
		}
	}
	return items
}

func runTry(app config.App, args []string) error {
	if len(args) > 0 && args[0] == "--profile" {
		return runTryProfile(app, args[1:])
	}
	kind, _, _, packages, err := parseInstallArgs(args)
	if err != nil {
		return err
	}
	if len(packages) == 0 {
		return fmt.Errorf("usage: nun try [--brew|--cask] <package>...")
	}
	action, err := ui.ShowPlan(ui.PlanView{
		Title:   "nun try",
		Summary: "Temporarily install packages and remember them for later.",
		Sections: []ui.PlanSection{
			{Title: "Temporary install", Items: describeTargets(kind, packages)},
			{Title: "Files to modify", Items: []string{"nun-trials.json"}},
		},
		Actions: []ui.PlanAction{ui.PlanApply, ui.PlanCancel},
	})
	if err != nil {
		return err
	}
	if action != ui.PlanApply {
		fmt.Println("aborted")
		return nil
	}
	return app.TryPackages(kind, packages)
}

func runTryProfile(app config.App, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("usage: nun try --profile [host]")
	}
	host := ""
	if len(args) == 1 {
		host = args[0]
	} else {
		hosts, err := app.HostNames()
		if err != nil {
			return err
		}
		host, err = ui.PickHost(hosts)
		if err != nil {
			return err
		}
		if host == "" {
			fmt.Println("aborted")
			return nil
		}
	}

	action, err := ui.ShowPlan(ui.PlanView{
		Title:   "Try profile",
		Summary: "Dry-run the host system build without installing anything.",
		Sections: []ui.PlanSection{
			{Title: "Command", Items: []string{"nun try --profile " + host}},
			{Title: "Host", Items: []string{host}},
			{Title: "Check", Items: []string{"nix build --dry-run path:<config-root>#darwinConfigurations." + host + ".system"}},
		},
		Actions: []ui.PlanAction{ui.PlanApply, ui.PlanCancel},
	})
	if err != nil {
		return err
	}
	if action != ui.PlanApply {
		fmt.Println("aborted")
		return nil
	}
	return app.TryProfile(host)
}

func describeTargets(kind config.InstallKind, packages []string) []string {
	items := make([]string, len(packages))
	for i, pkg := range packages {
		items[i] = string(kind) + " " + pkg
	}
	return items
}

func describeInstallTargets(targets []config.InstallTarget) []string {
	items := make([]string, len(targets))
	for i, target := range targets {
		targetName := "current host"
		if target.PackageSet != "" {
			targetName = target.PackageSet
		}
		items[i] = fmt.Sprintf("%s %s -> %s", target.Kind, target.Package, targetName)
	}
	return items
}

func describeWrites(writes []config.PendingWrite) []string {
	items := make([]string, len(writes))
	for i, write := range writes {
		items[i] = write.RelativePath
	}
	return items
}

func runInstall(app config.App, args []string) error {
	kind, set, interactive, packages, err := parseInstallArgs(args)
	if err != nil {
		return err
	}
	defaults, err := app.InstallDefaults()
	if err != nil {
		return err
	}
	fromTry := len(packages) == 0
	if fromTry && len(defaults.TryPackages) == 0 {
		return fmt.Errorf("no packages supplied and try list is empty")
	}

	req := config.InstallRequest{
		Kind:        kind,
		Packages:    packages,
		PackageSet:  set,
		Interactive: interactive,
		FromTryList: fromTry,
	}
	if fromTry {
		action, err := ui.ShowPlan(ui.PlanView{
			Title:   "Install trial packages",
			Summary: "The packages in nun-trials.json will be made permanent.",
			Sections: []ui.PlanSection{
				{Title: "Trial packages", Items: describeInstallTargets(defaults.TryPackages)},
			},
			Actions: []ui.PlanAction{ui.PlanApply, ui.PlanInteractive, ui.PlanCancel},
		})
		if err != nil {
			return err
		}
		if action == ui.PlanCancel {
			fmt.Println("aborted")
			return nil
		}
		if action == ui.PlanInteractive {
			req, err = interactiveTryInstall(defaults)
			if err != nil {
				return err
			}
		}
	} else if interactive {
		req, err = interactiveInstall(req, defaults)
		if err != nil {
			return err
		}
	}

	plan, err := app.PlanInstall(req)
	if err != nil {
		return err
	}
	action, err := ui.ShowPlan(ui.PlanView{
		Title:   "nun install",
		Summary: "This will temporarily install packages now and edit package lists for the next rebuild.",
		Sections: []ui.PlanSection{
			{Title: "Temporary install", Items: describeInstallTargets(plan.Targets)},
			{Title: "Files to modify", Items: describeWrites(plan.Writes)},
		},
		Notes:   plan.Notes,
		Actions: []ui.PlanAction{ui.PlanApply, ui.PlanCancel},
	})
	if err != nil {
		return err
	}
	if action != ui.PlanApply {
		fmt.Println("aborted")
		return nil
	}
	return app.ApplyInstall(plan, os.Stdout)
}

func parseInstallArgs(args []string) (config.InstallKind, string, bool, []string, error) {
	kind := config.InstallNix
	set := ""
	interactive := false
	var packages []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-i", "--interactive":
			interactive = true
		case "--brew":
			kind = config.InstallBrew
		case "--cask":
			kind = config.InstallCask
		case "--set":
			i++
			if i >= len(args) {
				return "", "", false, nil, fmt.Errorf("--set requires a package set")
			}
			set = args[i]
		default:
			if strings.HasPrefix(arg, "-") {
				return "", "", false, nil, fmt.Errorf("unknown install flag: %s", arg)
			}
			packages = append(packages, arg)
		}
	}
	return kind, set, interactive, packages, nil
}

func interactiveInstall(req config.InstallRequest, defaults config.InstallDefaults) (config.InstallRequest, error) {
	kind, err := ui.Choice("Kind? [nix/brew/cask]", "nix", "brew", "cask")
	if err != nil {
		return req, err
	}
	req.Kind = config.InstallKind(kind)
	if req.Kind == config.InstallNix {
		set, err := ui.Prompt("Package set (blank for current host)", req.PackageSet)
		if err != nil {
			return req, err
		}
		req.PackageSet = set
	} else {
		set, err := ui.Prompt("Homebrew target (blank for current host, darwin for global)", req.PackageSet)
		if err != nil {
			return req, err
		}
		req.PackageSet = set
	}
	_ = defaults
	return req, nil
}

func interactiveTryInstall(defaults config.InstallDefaults) (config.InstallRequest, error) {
	var targets []config.InstallTarget
	for _, target := range defaults.TryPackages {
		keep, err := ui.Choice("Install "+string(target.Kind)+" "+target.Package+" permanently? [Y/n]", "y", "", "n")
		if err != nil {
			return config.InstallRequest{}, err
		}
		if keep == "n" {
			continue
		}
		if target.Kind == config.InstallNix {
			set, err := ui.Prompt("Package set for "+target.Package+" (blank for current host)", target.PackageSet)
			if err != nil {
				return config.InstallRequest{}, err
			}
			target.PackageSet = set
		} else {
			set, err := ui.Prompt("Homebrew target for "+target.Package+" (blank for current host, darwin for global)", target.PackageSet)
			if err != nil {
				return config.InstallRequest{}, err
			}
			target.PackageSet = set
		}
		targets = append(targets, target)
	}
	if len(targets) == 0 {
		return config.InstallRequest{}, fmt.Errorf("no packages selected")
	}
	return config.InstallRequest{Targets: targets}, nil
}

func runHosts(app config.App, args []string) error {
	if len(args) == 0 {
		return app.PrintCurrentHost(os.Stdout)
	}
	if args[0] != "new" {
		return fmt.Errorf("unknown hosts command: %s", args[0])
	}

	input, err := app.NewHostDefaults()
	if err != nil {
		return err
	}
	result, err := ui.NewHost(input)
	if err != nil {
		return err
	}
	if result.Aborted {
		fmt.Println("aborted")
		return nil
	}
	if result.Cancelled {
		return nil
	}

	plan, err := app.PlanNewHost(config.NewHostRequest{
		Name:        result.Name,
		User:        result.User,
		System:      result.System,
		PackageSets: result.PackageSets,
	})
	if err != nil {
		return err
	}
	config.PrintHostPlan(os.Stdout, plan)
	ok, err := ui.Confirm("Apply these changes?")
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("aborted")
		return nil
	}
	return app.ApplyWrites(plan.Writes, os.Stdout)
}
