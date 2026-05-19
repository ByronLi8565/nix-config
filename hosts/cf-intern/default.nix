{mkDarwinHost, ...}:
mkDarwinHost {
  name = "cf-intern";
  user = "byron";
  system = "aarch64-darwin";
  configRoot = "/Users/byron/nix-config";

  extraModules = [
    ./packages.nix
    ({lib, ...}: {
      nix-homebrew.enable = lib.mkForce false;
      homebrew.enable = lib.mkForce false;
    })
  ];
}
