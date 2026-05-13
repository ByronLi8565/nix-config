{
  config,
  pkgs,
  ...
}: {
  environment.shells = [pkgs.fish];
  environment.systemPackages = import ../../package-sets/darwin.nix pkgs;

  programs.fish.enable = true;
  users.users.${config.system.primaryUser}.shell = pkgs.fish;
}
