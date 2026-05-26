{
  config,
  inputs,
  pkgs,
  ...
}: let
  user = config.system.primaryUser;
  configRoot = config.environment.variables.NUN_CONFIG_ROOT;
in {
  home-manager.useGlobalPkgs = true;
  home-manager.useUserPackages = true;

  home-manager.users.${user} = {
    imports = [inputs.hunk.homeManagerModules.default];

    home.stateVersion = "24.05";
    home.username = user;
    home.homeDirectory = config.users.users.${user}.home;
    home.sessionVariables.NUN_CONFIG_ROOT = configRoot;
    home.file.".config/zellij/plugins/zjstatus.wasm".source = "${pkgs.zjstatus}/bin/zjstatus.wasm";

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

    programs.direnv = {
      enable = true;
      nix-direnv.enable = true;
    };

    programs.zoxide = {
      enable = true;
      options = ["--no-cmd"];
    };

    programs.hunk = {
      enable = true;
      settings = {
        theme = "gruvbox";
      };
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
  };
}
