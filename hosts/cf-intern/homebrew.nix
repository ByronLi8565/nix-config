{
  config,
  cf-tap ? null,
  pkgs,
  lib,
  ...
}:
# NOTE: cf-tap requires SSH access to gitlab.cfdata.org.
# If cf-tap is not available (e.g., during initial setup), you need to either:
# 1. Run the setup script first to configure SSH, then rebuild
# 2. Or manually add the tap after SSH is configured:
#    brew tap cloudflare/engineering ssh://git@gitlab.cfdata.org/cloudflare/pkg/homebrew.git
#    brew install cfsetup cf-paste cf-yubikey-agent cloudflare-certs docker-credential-cloudflared cf-k8s-tools
lib.mkIf (cf-tap != null) {
  nix-homebrew.taps = {
    "cloudflare/engineering" = cf-tap;
  };

  homebrew = {
    taps = [
      "cloudflare/engineering"
      "hashicorp/tap"
    ];

    brews = [
      # Cloudflare tools from cf-tap
      "cfsetup"
      "cf-paste"
      "cf-yubikey-agent"
      "cloudflare-certs"
      "docker-credential-cloudflared"
      "cf-k8s-tools"

      # HashiCorp vault
      "hashicorp/tap/vault"
    ];

    # Docker Desktop for macOS
    casks = [
      "docker-desktop"
    ];
  };
}
