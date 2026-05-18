{
  buildGoModule,
  configRoot ? "/Users/spheal/nix-config",
}:

buildGoModule {
  pname = "nun";
  version = "0.1.0";

  src = ./.;
  vendorHash = "sha256-5kUNTqC9Wy5R1FptX7PPX7BpSWz8BGXcRzaY2qn/Md8=";

  ldflags = [
    "-X"
    "nun/internal/config.defaultConfigRoot=${configRoot}"
  ];

  subPackages = ["cmd/nun"];
}
