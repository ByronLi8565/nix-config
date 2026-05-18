{
  # Determinate Nix manages the Nix daemon. Keep nix-darwin from trying to
  # activate or overwrite daemon-level Nix configuration.
  nix.enable = false;
}
