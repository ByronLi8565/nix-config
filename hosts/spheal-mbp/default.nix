{mkDarwinHost, ...}:
mkDarwinHost {
  name = "spheal-mbp";
  user = "spheal";
  system = "aarch64-darwin";

  extraModules = [
    ./homebrew.nix
    ./packages.nix
  ];
}
