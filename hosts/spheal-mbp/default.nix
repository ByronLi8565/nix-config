{mkDarwinHost, ...}:
mkDarwinHost {
  name = "spheal-mbp";
  user = "spheal";
  system = "aarch64-darwin";
  configRoot = "/Users/spheal/nix-config";

  extraModules = [
    ./homebrew.nix
    ./packages.nix
  ];
}
