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

  outputs = inputs: let
    inherit (builtins) attrNames listToAttrs readDir;

    mkDarwinHost = import ./lib/mk-darwin-host.nix inputs;

    hostNames = attrNames (readDir ./hosts);
    hosts = listToAttrs (map (name: {
        inherit name;
        value = import ./hosts/${name} {inherit inputs mkDarwinHost;};
      })
      hostNames);
  in {
    darwinConfigurations = hosts;
  };
}
