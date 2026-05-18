# nun

`nun` is a small Go CLI for this Nix config. It uses Bubble Tea for terminal UI
commands and keeps Nix expression parsing in `internal/nixexpr`.

## Build

From this directory:

```sh
go build ./cmd/nun
```

From the repo root, build the Nix package:

```sh
nix build --impure --expr 'let pkgs = import <nixpkgs> {}; in pkgs.callPackage ./bin/nun {}'
```

## Run

From this directory:

```sh
go run ./cmd/nun --help
go run ./cmd/nun hosts
go run ./cmd/nun packages
go run ./cmd/nun hosts new
go run ./cmd/nun try ripgrep
go run ./cmd/nun try --profile spheal-mbp
go run ./cmd/nun install ripgrep
go run ./cmd/nun install --set global ripgrep
go run ./cmd/nun install --brew tcl-tk
go run ./cmd/nun install --cask nikitabobko/tap/aerospace
go run ./cmd/nun link
```

`nun try` temporarily installs packages with `nix profile install` or `brew`
and records them in `nun-trials.json`. `nun try --profile [host]` dry-runs the
host flake build without installing anything; omit the host to choose one
interactively. `nun install` does the temporary
install and edits the relevant package list after confirmation. With no package
arguments, `nun install` reads the try list and asks before making those
packages permanent; choose `i` at that prompt to review packages interactively.

After a Nix build from the repo root:

```sh
./result/bin/nun --help
```

## Test

From this directory:

```sh
go test ./...
```
