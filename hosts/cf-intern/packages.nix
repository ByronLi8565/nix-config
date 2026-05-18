{pkgs, ...}: {
  home-manager.users.byron.home.packages =
    builtins.concatLists [
      (import ../../package-sets/darwin.nix pkgs).nixPackages
      (import ../../package-sets/global.nix pkgs)
    ];
}
