{
  aerospace-tap,
  config,
  homebrew-cask,
  homebrew-core,
  ...
}: {
  nix-homebrew = {
    enable = true;
    enableRosetta = true;
    autoMigrate = true;
    user = config.system.primaryUser;
    taps = {
      "nikitabobko/tap" = aerospace-tap;
      "homebrew/homebrew-core" = homebrew-core;
      "homebrew/homebrew-cask" = homebrew-cask;
    };
  };

  homebrew = {
    enable = true;
    taps = builtins.attrNames config.nix-homebrew.taps;
    onActivation = {
      cleanup = "zap";
      autoUpdate = true;
      upgrade = true;
    };
  };
}
