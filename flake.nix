{
  description = "nix-darwin system configuration";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

    nix-darwin = {
      url = "github:LnL7/nix-darwin";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    home-manager = {
      url = "github:nix-community/home-manager";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    nix-homebrew.url = "github:zhaofengli/nix-homebrew";
    homebrew-core = {
      url = "github:homebrew/homebrew-core";
      flake = false;
    };
    homebrew-cask = {
      url = "github:homebrew/homebrew-cask";
      flake = false;
    };
    aerospace-tap = {
      url = "github:nikitabobko/homebrew-tap";
      flake = false;
    };
  };
  outputs = inputs @ {
    self,
    nix-darwin,
    nixpkgs,
    home-manager,
    nix-homebrew,
    homebrew-core,
    homebrew-cask,
    aerospace-tap,
  }: let
    hostName = "spheal-mbp";
    darwinSystem = nix-darwin.lib.darwinSystem {
      modules = [
        nix-homebrew.darwinModules.nix-homebrew
        {
          nix-homebrew = {
            enable = true;
            enableRosetta = true;
            autoMigrate = true;
            user = "spheal";
            taps = {
              "nikitabobko/tap" = aerospace-tap;
              "homebrew/homebrew-core" = homebrew-core;
              "homebrew/homebrew-cask" = homebrew-cask;
            };
          };
        }
        (
          {config, ...}: {
            homebrew.taps = builtins.attrNames config.nix-homebrew.taps;
          }
        )

        (
          {pkgs, ...}: {
            networking.hostName = hostName;
            networking.localHostName = hostName;
            networking.computerName = hostName;
            nixpkgs.hostPlatform = "aarch64-darwin";
            nixpkgs.config.allowUnfree = true;
            environment.systemPackages = with pkgs; [
              yabai
              mas
              fish
            ];
            homebrew = {
              enable = true;
              casks = [
                "nikitabobko/tap/aerospace"
                "visual-studio-code"
                "docker-desktop"
                "ghostty"
                "unity-hub"
                "zoom"
                "slack"
                "zed"
                "discord"
                "claude"
                "steam"
                "sol"
                "zen"
                "skim"
                "arc"
                "spotify"
              ];

              brews = [
                "tcl-tk"
              ];

              onActivation = {
                cleanup = "zap";
                autoUpdate = true;
                upgrade = true;
              };
            };
            nix.package = pkgs.nix;
            nix.settings = {
              experimental-features = "nix-command flakes";
              substituters = [
                "https://cache.nixos.org/"
                "https://jrestivo.cachix.org"
              ];
              trusted-public-keys = [
                "cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY="
                "jrestivo.cachix.org-1:+jSOsXAAOEjs+DLkybZGQEEIbPG7gsKW1hPwseu03OE="
              ];
              trusted-users = [
                "root"
                "spheal"
              ];
            };
            environment.shells = [pkgs.fish];
            system.primaryUser = "spheal";
            system.configurationRevision = self.rev or self.dirtyRev or null;
            system.stateVersion = 5;
            system.defaults = {
              dock.autohide = true;
              NSGlobalDomain.InitialKeyRepeat = 12;
              NSGlobalDomain.KeyRepeat = 2;
              NSGlobalDomain.NSAutomaticCapitalizationEnabled = false;
              NSGlobalDomain.NSAutomaticInlinePredictionEnabled = false;
              NSGlobalDomain.NSAutomaticSpellingCorrectionEnabled = false;
              NSGlobalDomain.NSAutomaticWindowAnimationsEnabled = false;
              dock.orientation = "right";
              dock.persistent-apps = [];
              dock.mru-spaces = true;
              finder.AppleShowAllExtensions = true;
              NSGlobalDomain.AppleShowAllExtensions = true;
            };
            security.pam.services.sudo_local.touchIdAuth = true;
            fonts.packages = with pkgs; [
              nerd-fonts.fira-code
            ];
            programs.fish.enable = true;
            users.users.spheal = {
              name = "spheal";
              home = "/Users/spheal";
              shell = pkgs.fish;
            };
          }
        )
        home-manager.darwinModules.home-manager
        {
          home-manager.useGlobalPkgs = true;
          home-manager.useUserPackages = true;
          home-manager.users."spheal" = {pkgs, ...}: {
            home.stateVersion = "24.05";
            home.username = "spheal";

            home.packages = import ./packages.nix pkgs;

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
                rerere = {
                  enabled = true;
                };
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
                rebuild = "sudo darwin-rebuild switch --flake /Users/spheal/nix-config#spheal-mbp";
                ls = "lsd -A";
              };

              interactiveShellInit = ''
                set -gx PATH $HOME/.nix-profile/bin /etc/profiles/per-user/$USER/bin $PATH
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
                  # Check if this is a login shell
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
              settings = {
                character = {
                  success_symbol = "[λ](bold green)";
                  error_symbol = "[λ](bold red)";
                };
              };
            };
          };
        }
      ];
    };
  in {
    darwinConfigurations = {
      ${hostName} = darwinSystem;
      spheal-mbp-9 = darwinSystem;
      spheal-mbp-10 = darwinSystem;
    };
  };
}
