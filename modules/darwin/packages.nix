{
  config,
  pkgs,
  ...
}: let
  darwinPackages = import ../../package-sets/darwin.nix pkgs;
in {
  environment.shells = [pkgs.fish];
  environment.systemPackages = darwinPackages.nixPackages;

  programs.fish.enable = true;
  users.users.${config.system.primaryUser}.shell = pkgs.fish;
}
