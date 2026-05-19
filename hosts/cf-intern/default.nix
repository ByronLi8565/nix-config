{mkDarwinHost, ...}:
mkDarwinHost {
  name = "cf-intern";
  user = "byron";
  system = "aarch64-darwin";
  configRoot = "/Users/byron/nix-config";

  extraModules = [
    ./packages.nix
    ./homebrew.nix
  ];
}
