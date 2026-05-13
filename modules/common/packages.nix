{
  config,
  pkgs,
  ...
}: {
  home-manager.users.${config.system.primaryUser}.home.packages =
    import ../../package-sets/global.nix pkgs;
}
