{buildGoModule}:

buildGoModule {
  pname = "nun";
  version = "0.1.0";

  src = ./.;
  vendorHash = "sha256-5kUNTqC9Wy5R1FptX7PPX7BpSWz8BGXcRzaY2qn/Md8=";

  subPackages = ["cmd/nun"];
}
