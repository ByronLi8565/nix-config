{
  pkgs,
  inputs,
  config,
  lib,
  ...
}: let
  user = config.system.primaryUser;
  homeDir = config.users.users.${user}.home;

  darwinPkgs = (import ../../package-sets/darwin.nix pkgs).nixPackages;
  globalPkgs = import ../../package-sets/global.nix pkgs;
in {
  # NOTE: Git configuration for Cloudflare should be set manually:
  #   git config --global user.email "byron@cloudflare.com"
  #   git config --global user.name "Your Name"

  home-manager.users.${user} = {
    home.packages = darwinPkgs ++ globalPkgs ++ (with pkgs; [
  helix
    ]);

  };
}
