{
  config,
  lib,
  pkgs,
  ...
}:

let
  # A hook's command with the shell's Go first on PATH, the shell's module
  # cache, and cgo off as in the shell. git runs hooks with the caller's
  # environment, so a commit from outside the shell (an editor, a GUI) would
  # otherwise find no `go`, which golangci-lint also shells out to, download
  # every module again, and look for a C compiler that is not there.
  withGo =
    name: command:
    lib.getExe (
      pkgs.writeShellApplication {
        inherit name;
        runtimeInputs = [ config.languages.go.package ];
        text = ''
          export CGO_ENABLED=0
          export GOPATH="''${GOPATH:-$(git rev-parse --show-toplevel)/.devenv/state/go}"
          ${command}
        '';
      }
    );
in
{
  # Harold is pure Go, built without cgo everywhere: here, in the hooks, in
  # the flake and in the Dockerfile. With it on, Go's net package wants a C
  # compiler, and a build here could differ from the ones shipped.
  env.CGO_ENABLED = "0";

  packages = with pkgs; [
    gh
    git
    inotify-tools
    jq
    skopeo
    # Drives Harold in a real terminal that an agent can read back
    # (`tmux capture-pane`), for checking the screen beyond the golden files.
    tmux
  ];

  # `harold` in the shell is the checkout's: the commands the docs give work
  # here as written, against the code being changed. Outside the shell,
  # `harold` is whatever is installed. Built and exec'd rather than `go run`,
  # so signals and the exit status reach Harold itself.
  scripts.harold = {
    description = "Run this checkout's harold";
    exec = ''
      go build -C "$DEVENV_ROOT" -o "$DEVENV_STATE/harold" ./cmd/harold
      exec "$DEVENV_STATE/harold" "$@"
    '';
  };

  # The dev loop: Harold rebuilt and restarted on every save, on the same
  # card. A restart is a SIGTERM, which Bubble Tea treats as a quit, so the
  # terminal is restored and the position written; `q` ends the loop and
  # forgets the position. Harold runs in the background of this script with
  # the terminal as its input: without job control, the script's background
  # jobs stay in the terminal's foreground process group, so it can read keys.
  scripts.harold-dev = {
    description = "Run harold, rebuilt and restarted on the same card on every save";
    exec = ''
      set -u
      root="$DEVENV_ROOT"
      state="$DEVENV_STATE/harold-dev.json"
      bin="$DEVENV_STATE/harold-dev"
      changed="$DEVENV_STATE/harold-dev.changed"
      rm -f "$changed"
      watch() {
        inotifywait -qq -r -e close_write,create,delete,move \
          --include '(\.go|go\.mod|go\.sum)$' "$root/cmd" "$root/internal" "$root/go.mod"
      }
      while true; do
        if ! go build -C "$root" -o "$bin" ./cmd/harold; then
          echo "harold-dev: the build failed; waiting for a change." >&2
          watch
          continue
        fi
        "$bin" --dev-state "$state" "$@" </dev/tty &
        harold=$!
        (watch && touch "$changed" && kill -TERM "$harold" 2>/dev/null) &
        watcher=$!
        wait "$harold"
        pkill -P "$watcher" 2>/dev/null
        kill "$watcher" 2>/dev/null
        wait "$watcher" 2>/dev/null
        if [ ! -e "$changed" ]; then
          rm -f "$state"
          break
        fi
        rm -f "$changed"
      done
    '';
  };

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
