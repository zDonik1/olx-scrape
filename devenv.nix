{ pkgs, ... }:

{
  packages = with pkgs; [
    gofumpt
    golangci-lint
  ];

  languages.go.enable = true;
}
