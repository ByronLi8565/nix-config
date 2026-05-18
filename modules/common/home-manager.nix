{
  config,
  pkgs,
  ...
}: let
  user = config.system.primaryUser;
  configRoot = config.environment.variables.NUN_CONFIG_ROOT;
in {
  home-manager.useGlobalPkgs = true;
  home-manager.useUserPackages = true;

  home-manager.users.${user} = {
    home.stateVersion = "24.05";
    home.username = user;
    home.homeDirectory = config.users.users.${user}.home;
    home.sessionVariables.NUN_CONFIG_ROOT = configRoot;
    home.file.".config/fish/fish_plugins".source = ../../dotfiles/fish/fish_plugins;
    home.file.".config/fish/functions/sesh.fish".source = ../../dotfiles/fish/functions/sesh.fish;
    home.file.".config/fish/completions/sesh.fish".source = ../../dotfiles/fish/completions/sesh.fish;

    programs.git = {
      enable = true;
      settings = {
        push = {autoSetupRemote = true;};
        core = {editor = "vim";};
        pull = {rebase = false;};
        user.name = "ByronLi8565";
        user.email = "byronli8565@gmail.com";
        aliases = {
          co = "checkout";
          br = "branch";
          ci = "commit";
          st = "status";
        };
        rerere.enabled = true;
      };
    };

    programs.fish = {
      enable = true;
      shellAbbrs = {
        ggg = {
          expansion = "git add -A && git commit -m '%'";
          setCursor = true;
        };
        v = "nvim";
      };

      shellAliases = {
        rebuild = "sudo darwin-rebuild switch --flake ${configRoot}#${config.networking.hostName}";
        ls = "lsd -A";
      };

      interactiveShellInit = ''
        set -gx PATH $HOME/.nix-profile/bin /etc/profiles/per-user/$USER/bin $PATH
        set -gx NUN_CONFIG_ROOT ${configRoot}
        eval (opam env)
      '';

      functions = {
        fish_greeting = "";
        z = {
          body = "__zoxide_z $argv && lsd";
          wraps = "__zoxide_z";
        };
      };
    };

    programs.direnv = {
      enable = true;
      nix-direnv.enable = true;
    };

    programs.zoxide = {
      enable = true;
      options = ["--no-cmd"];
    };

    programs.zsh = {
      enable = true;
      initContent = ''
        if [[ $(ps -o comm= -p $PPID) != "fish" && -z ''${BASH_EXECUTION_STRING} ]]; then
          if [[ -o login ]]; then
            exec ${pkgs.fish}/bin/fish --login
          else
            exec ${pkgs.fish}/bin/fish
          fi
        fi
      '';
    };

    programs.starship = {
      enable = true;
      settings.character = {
        success_symbol = "[λ](bold green)";
        error_symbol = "[λ](bold red)";
      };
    };
  };
}
