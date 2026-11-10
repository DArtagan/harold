{
  config,
  lib,
  pkgs,
  ...
}:

let
  # A hook's command with the shell's Go first on PATH. git runs hooks with the
  # caller's environment, so a commit from outside the shell (an editor, a GUI)
  # would otherwise find no `go`, which golangci-lint also shells out to.
  withGo =
    name: command:
    lib.getExe (
      pkgs.writeShellApplication {
        inherit name;
        runtimeInputs = [ config.languages.go.package ];
        text = command;
      }
    );
in
{
  packages = with pkgs; [
    gh
    git
    jq
    skopeo
  ];

  # nixpkgs' default Go, deliberately not pinned: devenv rebuilds its Go tools
  # (gopls, staticcheck, delve...) from source for any other version, which no
  # binary cache holds, so every new shell and every CI run would compile them.
  # The Dockerfile's builder image matches this Go's minor version.
  languages.go.enable = true;

  # treefmt runs on commit, under `devenv test` and by hand: entering the shell
  # does not rewrite files.
  tasks."devenv:treefmt:run".before = lib.mkForce [ ];

  treefmt = {
    enable = true;

    config = {
      settings.global.excludes = [
        "devenv.lock"
        "go.sum"
        "LICENSE"
        # Written by release-please.
        "CHANGELOG.md"
      ];

      programs = {
        deadnix.enable = true;
        gofmt.enable = true;
        jsonfmt.enable = true;
        nixfmt.enable = true;
        statix.enable = true;
        taplo.enable = true;
        yamlfmt.enable = true;
      };

      # Leave blank lines as written, and do not re-fold `>` scalars.
      programs.yamlfmt.settings.formatter = {
        retain_line_breaks = true;
        scan_folded_as_literal = true;
      };

      settings.formatter = {
        deadnix.priority = 0;
        statix.priority = 0;
        nixfmt.priority = 1;

        # The types no formatter above covers: Markdown, the Dockerfile, .envrc.
        whitespace = {
          command = lib.getExe (
            pkgs.writeShellApplication {
              name = "whitespace";
              runtimeInputs = with pkgs; [
                coreutils
                diffutils
                gnused
              ];
              text = ''
                for file in "$@"; do
                  tmp="$(mktemp)"
                  printf '%s\n' "$(sed -e 's/[[:space:]]*$//' "$file")" > "$tmp"
                  # Only when something changed, so --fail-on-change stays honest.
                  cmp -s "$tmp" "$file" || cat "$tmp" > "$file"
                  rm -f "$tmp"
                done
              '';
            }
          );
          includes = [ "*" ];
          # Golden files are the terminal's exact output, trailing spaces
          # included; trimming them would fail every snapshot test.
          excludes = [ "*.golden" ];
          priority = 2;
        };
      };
    };
  };

  git-hooks.hooks = {
    treefmt.enable = true;

    # release-please derives every version and changelog entry from these.
    convco.enable = true;

    golangci-lint = {
      enable = true;
      name = "golangci-lint";
      entry = "${withGo "golangci-lint" "${lib.getExe pkgs.golangci-lint} run ./..."}";
      files = "\\.go$|^go\\.(mod|sum)$|^\\.golangci\\.yaml$";
      pass_filenames = false;
    };

    go-test = {
      enable = true;
      name = "go test";
      entry = "${withGo "go-test" "go test ./..."}";
      files = "\\.go$|^go\\.(mod|sum)$|\\.golden$|(^|/)testdata/";
      pass_filenames = false;
    };
  };
}
