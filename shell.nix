{pkgs ? import <nixpkgs> {}}:
pkgs.buildFHSEnv {
  name = "binstall-env";
  targetPkgs = pkgs: [
    pkgs.cargo-binstall
  ];
  runScript = "fish";
}
