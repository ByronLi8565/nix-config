{mkDarwinHost, ...}:
mkDarwinHost {
  name = "cf-intern";
  user = "byron";
  system = "aarch64-darwin";

  extraModules = [
    ./packages.nix
  ];
}
