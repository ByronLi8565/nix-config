{pkgs, ...}: {
  home-manager.users.spheal.home.packages =
    import ../../package-sets/development.nix pkgs;
}
