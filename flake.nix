{
  description = "Harold: triage a Todoist list one task at a time. Not created by, affiliated with, or supported by Todoist.";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    # `...` because Nix also passes `self`, which nothing here uses; deadnix
    # removes an unused `self`, and without `...` the flake then fails.
    { nixpkgs, ... }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forEach = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      packages = forEach (pkgs: {
        # Go's minor version is the devenv shell's and the Dockerfile's, so the
        # package installed is built by the compiler the tests ran under;
        # nixos-unstable's default runs a minor ahead of devenv's. Moved
        # together with those two.
        default = pkgs.buildGo126Module rec {
          pname = "harold";
          # Written by release-please; see docs/releasing.md.
          version = "0.0.0"; # x-release-please-version

          src = pkgs.lib.fileset.toSource {
            root = ./.;
            fileset = pkgs.lib.fileset.unions [
              ./go.mod
              ./go.sum
              ./cmd
              ./internal
            ];
          };

          # Changes with every change to go.sum; CI's flake job builds this
          # package so a stale hash fails there, not at install time.
          vendorHash = "sha256-IP2hW6LNw8YwN2BSiMbGqXw0ZT6KsqqFGOUYLa4mmyY=";

          # Pure Go, as everywhere else Harold is built (devenv.nix).
          env.CGO_ENABLED = 0;

          subPackages = [ "cmd/harold" ];
          ldflags = [
            "-s"
            "-w"
            "-X main.version=${version}"
          ];

          meta = {
            description = "Harold: triage a Todoist list one task at a time. Not created by, affiliated with, or supported by Todoist";
            homepage = "https://github.com/DArtagan/harold";
            license = pkgs.lib.licenses.mit;
            mainProgram = "harold";
          };
        };
      });
    };
}
