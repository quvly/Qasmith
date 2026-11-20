#!/bin/sh
set -eu

root=$(cd "$(dirname "$0")/../.." && pwd)
fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT
case "$(uname -s)" in
    Darwin) os=darwin ;;
    Linux) os=linux ;;
    *) exit 1 ;;
esac
case "$(uname -m)" in
    arm64|aarch64) arch=arm64 ;;
    x86_64|amd64) arch=amd64 ;;
    *) exit 1 ;;
esac
asset="qasmith-$os-$arch"
version=$("$root/qasmith" version)
export QASMITH_TEST_TAG="${version#Qasmith }"
cp "$root/qasmith" "$fixture/$asset"
(cd "$fixture" && shasum -a 256 "$asset" | awk '{print $1}' > "$asset.sha256")
cat > "$fixture/curl" <<'MOCK'
#!/bin/sh
set -eu
while [ "$#" -gt 0 ]; do
    case "$1" in
        --output) output=$2; shift ;;
        "https://github.com/quvly/Qasmith/releases/download/$QASMITH_TEST_TAG/"*) asset=${1##*/} ;;
    esac
    shift
done
cp "$QASMITH_FIXTURE/$asset" "$output"
MOCK
chmod +x "$fixture/curl"
export QASMITH_FIXTURE="$fixture"
export PATH="$fixture:$PATH"
destination="$fixture/bin with spaces"
sh "$root/scripts/install.sh" --version "$QASMITH_TEST_TAG" --bin-dir "$destination"
test "$("$destination/qasmith" version)" = "$version"
sh "$root/scripts/install.sh" --version "$QASMITH_TEST_TAG" --bin-dir "$destination"
printf '%s\n' 'bad download' > "$fixture/$asset"
if sh "$root/scripts/install.sh" --version "$QASMITH_TEST_TAG" --bin-dir "$destination"; then
    echo 'Installer accepted a corrupt download' >&2
    exit 1
fi
test "$("$destination/qasmith" version)" = "$version"
