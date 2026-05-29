#!/bin/sh
set -eu

REPO="romerramos/previously_on"
BIN_NAME="previously-on"
VERSION="latest"
INSTALL_DIR="${PREVIOUSLY_ON_INSTALL_DIR:-}"
SKIP_CONNECT="${PREVIOUSLY_ON_SKIP_CONNECT:-}"

usage() {
  cat <<'EOF'
Install previously-on from GitHub Releases.

Usage:
  install.sh [--version v0.1.0] [--dir /usr/local/bin] [--no-connect]

Environment:
  PREVIOUSLY_ON_INSTALL_DIR  Install destination when --dir is not provided.
  PREVIOUSLY_ON_SKIP_CONNECT Skip AI provider setup when set to 1.
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      VERSION="${2:-}"
      if [ -z "$VERSION" ]; then
        echo "--version requires a value" >&2
        exit 1
      fi
      shift 2
      ;;
    --dir)
      INSTALL_DIR="${2:-}"
      if [ -z "$INSTALL_DIR" ]; then
        echo "--dir requires a value" >&2
        exit 1
      fi
      shift 2
      ;;
    --no-connect)
      SKIP_CONNECT="1"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "required command not found: $1" >&2
    exit 1
  fi
}

need curl
need tar

os="$(uname -s)"
arch="$(uname -m)"

case "$os" in
  Darwin) os="Darwin" ;;
  Linux) os="Linux" ;;
  *)
    echo "unsupported OS: $os" >&2
    exit 1
    ;;
esac

case "$arch" in
  x86_64|amd64) arch="x86_64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

if [ "$VERSION" = "latest" ]; then
  release_url="https://github.com/$REPO/releases/latest"
else
  release_url="https://github.com/$REPO/releases/tag/$VERSION"
fi

resolved_url="$(curl -fsSL -o /dev/null -w '%{url_effective}' "$release_url")"
VERSION="${resolved_url##*/}"

archive="previously-on_${os}_${arch}.tar.gz"
base_url="https://github.com/$REPO/releases/download/$VERSION"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT INT TERM

if [ -z "$INSTALL_DIR" ]; then
  if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="$HOME/.local/bin"
  fi
fi

echo "Installing $BIN_NAME $VERSION for $os/$arch"
echo "Downloading $base_url/$archive"

curl -fsSL "$base_url/$archive" -o "$tmpdir/$archive"
curl -fsSL "$base_url/checksums.txt" -o "$tmpdir/checksums.txt"

if command -v sha256sum >/dev/null 2>&1; then
  (cd "$tmpdir" && grep "  $archive$" checksums.txt | sha256sum -c -)
elif command -v shasum >/dev/null 2>&1; then
  (cd "$tmpdir" && grep "  $archive$" checksums.txt | shasum -a 256 -c -)
else
  echo "warning: sha256sum or shasum not found; skipping checksum verification" >&2
fi

tar -xzf "$tmpdir/$archive" -C "$tmpdir"
mkdir -p "$INSTALL_DIR"
cp "$tmpdir/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
chmod 0755 "$INSTALL_DIR/$BIN_NAME"

echo "Installed $BIN_NAME to $INSTALL_DIR/$BIN_NAME"
if [ "$SKIP_CONNECT" != "1" ]; then
  echo
  echo "Set up an AI provider now, or cancel and run '$BIN_NAME connect' later."
  if ! "$INSTALL_DIR/$BIN_NAME" connect; then
    echo "AI provider setup skipped. Run '$BIN_NAME connect' when you are ready." >&2
  fi
fi

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) echo "Warning: $INSTALL_DIR is not in your PATH" >&2 ;;
esac

cat <<EOF

Next:
  $BIN_NAME summary
  $BIN_NAME connect
  $BIN_NAME install nvim --strategy lazy
EOF
