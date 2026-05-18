package config

import "strings"

type HostDefaultInput struct {
	Name             string
	User             string
	System           string
	ConfigRoot       string
	HasPackageModule bool
}

func RenderHostDefault(input HostDefaultInput) string {
	extraModules := ""
	if input.HasPackageModule {
		extraModules = "    ./packages.nix\n"
	}
	configRoot := input.ConfigRoot
	if configRoot == "" {
		configRoot = "/Users/" + input.User + "/nix-config"
	}
	return `{mkDarwinHost, ...}:
mkDarwinHost {
  name = "` + input.Name + `";
  user = "` + input.User + `";
  system = "` + input.System + `";
  configRoot = "` + configRoot + `";

  extraModules = [
` + extraModules + `  ];
}
`
}

func RenderHostPackages(user string, packageSets []string) string {
	imports := make([]string, len(packageSets))
	for i, set := range packageSets {
		imports[i] = "      " + hostPackageSetExpr(set)
	}
	return `{pkgs, ...}: {
  home-manager.users.` + user + `.home.packages =
    builtins.concatLists [
` + strings.Join(imports, "\n") + `
    ];
}
`
}

func hostPackageSetExpr(set string) string {
	expr := "import ../../package-sets/" + set + ".nix pkgs"
	if set == "darwin" {
		return "(" + expr + ").nixPackages"
	}
	return "(" + expr + ")"
}
