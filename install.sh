#!/usr/bin/env bash
# installer: compiles the Go binary locally and installs it into a
# user-writable prefix so sudo is never needed.

set -euo pipefail

APP_NAME="cawder"

if ! command -v go >/dev/null 2>&1; then
  echo "error: go is required but not installed or not on PATH" >&2
  exit 1
fi

on_path() { case ":${PATH}:" in *":$1:"*) return 0 ;; *) return 1 ;; esac; }
writable_or_creatable() {
  [ -d "$1" ] && [ -w "$1" ] && return 0
  [ ! -e "$1" ] && mkdir -p "$1" 2>/dev/null && return 0
  return 1
}

if [ -n "${PREFIX:-}" ]; then
  bindir="${PREFIX}/bin"
else
  bindir=""
  for d in "$HOME/.local/bin" "/usr/local/bin" "$HOME/bin"; do
    if on_path "$d" && writable_or_creatable "$d"; then
      bindir="$d"; break
    fi
  done
  [ -z "$bindir" ] && bindir="$HOME/.local/bin"
fi

echo "Compiling ${APP_NAME}..."
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

go build -o "${tmp}/${APP_NAME}" ./cmd/cawder

install -d "${bindir}" 2>/dev/null \
  || { echo "error: cannot create ${bindir} - specify a writable PREFIX" >&2; exit 1; }
install -m 0755 "${tmp}/${APP_NAME}" "${bindir}/${APP_NAME}" 2>/dev/null \
  || { echo "error: cannot write to ${bindir}/${APP_NAME} - specify a writable PREFIX" >&2; exit 1; }

echo "Installed: ${bindir}/${APP_NAME}"

if ! on_path "$bindir"; then
  line="export PATH=\"${bindir}:\$PATH\""
  marker="# ${APP_NAME}-path"
  touched=0
  for rc in "$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.profile" "$HOME/.zshrc"; do
    [ -f "$rc" ] || continue
    grep -qsF "$marker" "$rc" && continue
    grep -qsF "${bindir}" "$rc" && continue
    printf '\n%s\n%s\n' "$marker" "$line" >> "$rc"
    echo "Added PATH entry to ${rc}"
    touched=1
  done

  echo ""
  if [ "$touched" = 1 ]; then
    echo "New shells will pick this up automatically."
    echo "For this shell, run:"
  else
    echo "${bindir} is not on PATH. Run this in your shell:"
  fi
  echo "  ${line}"
fi
