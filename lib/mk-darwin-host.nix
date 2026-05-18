inputs: {
  name,
  user,
  system ? "aarch64-darwin",
  configRoot ? "/Users/${user}/nix-config",
  extraModules ? [],
}: let
  inherit (inputs) home-manager nix-darwin nix-homebrew self;
in
  nix-darwin.lib.darwinSystem {
    specialArgs =
      inputs
      // {
        inherit inputs;
      };

    modules =
      [
        nix-homebrew.darwinModules.nix-homebrew
        home-manager.darwinModules.home-manager

        {
          networking.hostName = name;
          networking.localHostName = name;
          networking.computerName = name;

          nixpkgs.hostPlatform = system;
          nixpkgs.config.allowUnfree = true;

          system.primaryUser = user;
          system.configurationRevision = self.rev or self.dirtyRev or null;
          system.stateVersion = 5;
          environment.variables.NUN_CONFIG_ROOT = configRoot;

          users.users.${user} = {
            name = user;
            home = "/Users/${user}";
          };
        }

        ../modules/common/home-manager.nix
        ../modules/common/nix.nix
        ../modules/common/packages.nix

        ../modules/darwin/defaults.nix
        ../modules/darwin/fonts.nix
        ../modules/darwin/homebrew.nix
        ../modules/darwin/packages.nix
        ../modules/darwin/security.nix
      ]
      ++ extraModules;
  }
