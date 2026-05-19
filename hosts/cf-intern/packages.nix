{
  pkgs,
  inputs,
  config,
  lib,
  ...
}: let
  user = config.system.primaryUser;
  homeDir = config.users.users.${user}.home;
in {
  # NOTE: Git configuration for Cloudflare should be set manually:
  #   git config --global user.email "byron@cloudflare.com"
  #   git config --global user.name "Your Name"

  home-manager.users.${user} = {
    home.packages =
      builtins.concatLists [
        builtins.concatLists [
          (import ../../package-sets/darwin.nix pkgs).nixPackages
          (import ../../package-sets/global.nix pkgs)
        ]
        (with pkgs; [
          cloudflared
          glab
          vault
          docker
          codex
        ])
      ];

    # Cloudflare Go proxy configuration
    home.sessionVariables = {
      GOPROXY = "https://athens.cfdata.org,direct";
      GONOSUMDB = "github.com/cloudflare,go.cfdata.org";
      GOPRIVATE = "";
    };

    # Shell configuration
    programs.fish.shellInit = ''
      # Source vault functions
      if test -f ~/.config/cloudflare/vault-funcs
        source ~/.config/cloudflare/vault-funcs
      end
    '';

    programs.bash.initExtra = ''
      # Source vault functions
      if [ -f ~/.config/cloudflare/vault-funcs ]; then
        source ~/.config/cloudflare/vault-funcs
      fi
    '';

    programs.zsh.initContent = lib.mkOrder 1000 ''
      # Source vault functions
      if [ -f ~/.config/cloudflare/vault-funcs ]; then
        source ~/.config/cloudflare/vault-funcs
      fi
    '';

    # SSH configuration for Cloudflare
    home.file = {
      ".ssh/cloudflare/config".source = ./ssh-config;
      ".config/cloudflare/vault-funcs".source = ./vault-funcs;
    };

    # Activation script for SSH setup
    home.activation.setupCfSSH = config.lib.dag.entryAfter ["writeBoundary"] ''
      # Create SSH directories if they don't exist
      $DRY_RUN_CMD mkdir -p $VERBOSE_ARG "${homeDir}/.ssh/cloudflare"
      $DRY_RUN_CMD chmod 0711 "${homeDir}/.ssh"
      $DRY_RUN_CMD chmod 0711 "${homeDir}/.ssh/cloudflare"

      # Generate SSH key if it doesn't exist
      if [[ ! -f "${homeDir}/.ssh/cloudflare/id_ed25519" ]]; then
        $DRY_RUN_CMD ${pkgs.openssh}/bin/ssh-keygen -t ed25519 \
          -C "${user}@cloudflare.com" \
          -f "${homeDir}/.ssh/cloudflare/id_ed25519" \
          -N ""
      fi

      # Add to SSH agent
      if [[ -f "${homeDir}/.ssh/cloudflare/id_ed25519" ]]; then
        $DRY_RUN_CMD ${pkgs.openssh}/bin/ssh-add --apple-use-keychain \
          "${homeDir}/.ssh/cloudflare/id_ed25519" 2>/dev/null || true
      fi
    '';

    # Add Cloudflare SSH config to main SSH config
    home.activation.setupSshConfig = config.lib.dag.entryAfter ["writeBoundary"] ''
      # Create main SSH config if it doesn't exist
      if [[ ! -f "${homeDir}/.ssh/config" ]]; then
        $DRY_RUN_CMD touch "${homeDir}/.ssh/config"
      fi

      # Add Cloudflare include if not present
      if ! grep -q "Include ~/.ssh/cloudflare/config" "${homeDir}/.ssh/config"; then
        $DRY_RUN_CMD echo "" >> "${homeDir}/.ssh/config"
        $DRY_RUN_CMD echo "# CLOUDFLARE SETUP https://gitlab.cfdata.org/cloudflare/devtools/setup-scripts/" >> "${homeDir}/.ssh/config"
        $DRY_RUN_CMD echo "Match all" >> "${homeDir}/.ssh/config"
        $DRY_RUN_CMD echo "    Include ~/.ssh/cloudflare/config" >> "${homeDir}/.ssh/config"
      fi
    '';
  };
}
