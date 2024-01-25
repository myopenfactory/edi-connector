{ pkgs ? import <nixpkgs> { } }:
with pkgs;
mkShell {
  buildInputs = [
    nixpkgs-fmt
    go
    goreleaser
    nsis
    protobuf
  ];

  shellHook = ''
    # ...
  '';
}
