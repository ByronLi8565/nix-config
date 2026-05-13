{
  bun,
  buildNpmPackage,
  configRoot ? "/Users/spheal/nix-config",
  lib,
  makeWrapper,
}:

buildNpmPackage {
  pname = "nun";
  version = "0.1.0";

  src = ./.;
  npmDepsHash = "sha256-RhnXIvoPy8ya84njheofhk5uNErEVivYlHs3Jm14oPQ=";
  nativeBuildInputs = [makeWrapper];
  dontNpmBuild = true;

  installPhase = ''
    runHook preInstall

    mkdir -p $out/lib/nun $out/bin
    cp -R package.json package-lock.json bun.lock tsconfig.json src node_modules $out/lib/nun
    makeWrapper ${bun}/bin/bun $out/bin/nun \
      --set-default NUN_CONFIG_ROOT ${lib.escapeShellArg configRoot} \
      --add-flags "run $out/lib/nun/src/main.ts"

    runHook postInstall
  '';
}
