# Dotfile symlinks managed by nun
# Each entry maps a source path (relative to dotfiles/) to a target path (relative to home)
[
  { source = "aerospace.toml"; target = ".aerospace.toml"; }
  { source = "skhdrc"; target = ".skhdrc"; }
  { source = "yabairc"; target = ".yabairc"; }
  { source = "ghostty/config"; target = ".config/ghostty/config"; }
  { source = "nvim/init.lua"; target = ".config/nvim/init.lua"; }
  { source = ".config/zellij/config.kdl"; target = ".config/zellij/config.kdl"; }
  { source = ".config/zellij/layouts/battlestation.kdl"; target = ".config/zellij/layouts/battlestation.kdl"; }
  { source = ".config/helix/config.toml"; target = ".config/helix/config.toml"; }
  { source = "fish/config.fish"; target = ".config/fish/config.fish"; }
  { source = "fish/fish_plugins"; target = ".config/fish/fish_plugins"; }
  { source = "fish/functions/sesh.fish"; target = ".config/fish/functions/sesh.fish"; }
  { source = "fish/completions/sesh.fish"; target = ".config/fish/completions/sesh.fish"; }
  { source = "starship.toml"; target = ".config/starship.toml"; }
  { source = ".config/opencode/opencode.json"; target = ".config/opencode/opencode.json"; }
]
