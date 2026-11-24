#!/bin/sh
set -eu

usage() {
    printf '%s\n' 'Usage: install.sh --version <latest|vX.Y.Z> [--bin-dir DIRECTORY]'
    printf '%s\n' ''
    printf '%s\n' 'The --version flag is required. Pass --version latest to install the most recent release,'
    printf '%s\n' 'or --version vX.Y.Z to pin a specific release.'
}

version=""
bin_dir="${HOME}/.local/bin"
while [ "$#" -gt 0 ]; do
    case "$1" in
        --version|--bin-dir)
            [ "$#" -ge 2 ] && [ -n "$2" ] || { usage >&2; exit 2; }
            case "$1" in
                --version) version=$2 ;;
                --bin-dir) bin_dir=$2 ;;
            esac
            shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) usage >&2; exit 2 ;;
    esac
done

if [ -z "$version" ]; then
    printf '%s\n' 'Error: --version is required. Pass --version latest or --version vX.Y.Z.' >&2
    usage >&2
    exit 2
fi

case "$(uname -s)" in
    Darwin) platform=darwin ;;
    Linux) platform=linux ;;
    *) printf '%s\n' 'This installer supports macOS and Linux.' >&2; exit 1 ;;
esac
case "$(uname -m)" in
    x86_64|amd64) architecture=amd64 ;;
    aarch64|arm64) architecture=arm64 ;;
    *) printf '%s\n' 'Qasmith supports x86-64 and ARM64.' >&2; exit 1 ;;
esac
base=https://github.com/quvly/Qasmith/releases
if [ "$version" = latest ]; then
    url="$base/latest/download"
elif printf '%s\n' "$version" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
    url="$base/download/$version"
else
    printf '%s\n' 'Version must be latest or vX.Y.Z.' >&2
    exit 2
fi
asset="qasmith-$platform-$architecture"
mkdir -p -- "$bin_dir"
bin_dir=$(cd -- "$bin_dir" && pwd -P)
if [ -d "$bin_dir/qasmith" ]; then
    printf '%s\n' "Destination is a directory: $bin_dir/qasmith" >&2
    exit 1
fi
stage=$(mktemp -d "$bin_dir/.qasmith.XXXXXXXX")
trap 'rm -rf -- "$stage"' EXIT
trap 'exit 1' HUP INT TERM

curl --fail --silent --show-error --location --proto '=https' --output "$stage/$asset.sha256" "$url/$asset.sha256"
curl --fail --silent --show-error --location --proto '=https' --output "$stage/$asset" "$url/$asset"
expected=$(awk -v name="$asset" 'NF == 1 || (NF == 2 && $2 == name) { print $1 }' "$stage/$asset.sha256")
if [ -z "$expected" ]; then
    printf '%s\n' 'Could not read expected SHA-256 from checksum file.' >&2
    exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "$stage/$asset" | awk '{print $1}')
else
    actual=$(shasum -a 256 "$stage/$asset" | awk '{print $1}')
fi
if [ -z "$actual" ] || [ "$expected" != "$actual" ]; then
    printf '%s\n' 'Qasmith download failed SHA-256 verification.' >&2
    exit 1
fi
chmod 755 "$stage/$asset"
"$stage/$asset" version
mv -f -- "$stage/$asset" "$bin_dir/qasmith"
printf 'Installed %s\n' "$bin_dir/qasmith"
case ":${PATH-}:" in
    *":$bin_dir:"*) ;;
    *) printf 'Add %s to PATH in your shell configuration.\n' "$bin_dir" ;;
esac
