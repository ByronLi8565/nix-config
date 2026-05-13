{pkgs, ...}: {
  nix.package = pkgs.nix;
  nix.settings = {
    experimental-features = "nix-command flakes";
    substituters = [
      "https://cache.nixos.org/"
      "https://jrestivo.cachix.org"
    ];
    trusted-public-keys = [
      "cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY="
      "jrestivo.cachix.org-1:+jSOsXAAOEjs+DLkybZGQEEIbPG7gsKW1hPwseu03OE="
    ];
    trusted-users = [
      "root"
      "spheal"
    ];
  };
}
