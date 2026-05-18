pkgs: {
  nixPackages = with pkgs; [
    fish
    sqlitebrowser
    mpv-unwrapped
    pngpaste
    mupdf
    zathura
  ];

  homebrewBrews = [];

  homebrewCasks = [
    "nikitabobko/tap/aerospace"
    "visual-studio-code"
    "ghostty"
    "zoom"
    "slack"
    "zed"
    "discord"
    "claude"
    "skim"
    "arc"
    "spotify"
  ];
}
