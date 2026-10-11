#!/usr/bin/env bash
#
# Install the llm-api-test CLI from GitHub releases.
#
#   curl -fsSL https://raw.githubusercontent.com/suanova/llm-api-test/main/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/suanova/llm-api-test/main/install.sh | bash -s -- v1.2.3
#
# The version argument accepts `latest` (the default), `v1.2.3`, or `1.2.3`.
#
# Environment overrides:
#   INSTALL_DIR  where to put the binary (default: ~/.local/bin, no sudo needed)
#   USE_SUDO     use sudo to write INSTALL_DIR (default: only for /usr/local/bin)
#   GITHUB_REPO  owner/name to download from (default: suanova/llm-api-test)
set -eu

GITHUB_REPO=${GITHUB_REPO:-suanova/llm-api-test}
CLI_FILENAME=llm-api-test
INSTALL_DIR=${INSTALL_DIR:-${HOME}/.local/bin}
HTTP_CLI=
TARGET_OS=
TARGET_ARCH=
TMP_ROOT=

die() {
    echo "error: $*" >&2
    exit 1
}

cleanup() {
    if [ -n "$TMP_ROOT" ]; then
        rm -rf "$TMP_ROOT"
    fi
}

detect_platform() {
    local os arch
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)
    case "$arch" in
        x86_64|amd64) arch=amd64 ;;
        aarch64|arm64) arch=arm64 ;;
    esac

    case "$os" in
        linux|darwin) ;;
        mingw*|msys*|cygwin*)
            die "unsupported shell: this is a Windows environment. Releases ship a
       llm-api-test_<version>_windows_${arch}.zip instead; download and unpack it
       from https://github.com/${GITHUB_REPO}/releases."
            ;;
        *) die "unsupported OS: $os (prebuilt binaries exist for linux and darwin)" ;;
    esac
    case "$arch" in
        amd64|arm64) ;;
        *) die "unsupported architecture: $(uname -m) (prebuilt binaries exist for amd64 and arm64)" ;;
    esac

    TARGET_OS=$os
    TARGET_ARCH=$arch
}

detect_http_client() {
    if command -v curl >/dev/null 2>&1; then
        HTTP_CLI=curl
    elif command -v wget >/dev/null 2>&1; then
        HTTP_CLI=wget
    else
        die "curl or wget is required"
    fi
}

# http_get URL -- print the body, fail on HTTP errors
http_get() {
    if [ "$HTTP_CLI" = curl ]; then
        curl -fsSL "$1"
    else
        wget -qO- "$1"
    fi
}

# download FILE_URL DEST
download() {
    if [ "$HTTP_CLI" = curl ]; then
        curl -fsSL "$1" -o "$2"
    else
        wget -q -O "$2" "$1"
    fi
}

# resolve_latest -- print the newest release tag
#
# The asset names embed the tag (unlike some installers, whose asset name is
# version-less and can be fetched from the /releases/latest/download/ alias),
# so the tag has to be looked up before an asset URL can be built.
resolve_latest() {
    local api json tag
    api="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
    json=$(http_get "$api") || die "failed to query $api"
    tag=$(printf '%s\n' "$json" | grep '"tag_name":' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    if [ -z "$tag" ]; then
        die "no releases found for ${GITHUB_REPO} (see https://github.com/${GITHUB_REPO}/releases)"
    fi
    echo "$tag"
}

setup_sudo() {
    # Only the system-wide directory needs root; ~/.local/bin does not.
    # An explicit USE_SUDO=true/false always wins.
    if [ -z "${USE_SUDO:-}" ]; then
        if [ "$INSTALL_DIR" = /usr/local/bin ] && [ "$(id -u)" -ne 0 ]; then
            USE_SUDO=true
        else
            USE_SUDO=false
        fi
    fi
}

run_as_root() {
    if [ "$USE_SUDO" = true ] && [ "$(id -u)" -ne 0 ]; then
        sudo "$@"
    else
        "$@"
    fi
}

install_binary() {
    local src=$1
    run_as_root mkdir -p "$INSTALL_DIR"
    run_as_root install -m 0755 "$src" "${INSTALL_DIR}/${CLI_FILENAME}"
    echo "Installed ${CLI_FILENAME} to ${INSTALL_DIR}/${CLI_FILENAME}"
}

note_path() {
    case ":${PATH}:" in
        *":${INSTALL_DIR}:"*) ;;
        *)
            echo
            echo "NOTE: ${INSTALL_DIR} is not in your PATH."
            echo "Add it with:  export PATH=\"${INSTALL_DIR}:\$PATH\""
            echo "(put that line in ~/.bashrc or ~/.zshrc to keep it)"
            ;;
    esac
}

main() {
    local version=${1:-latest}

    trap cleanup EXIT
    detect_http_client
    detect_platform
    setup_sudo

    if [ "$version" = latest ]; then
        version=$(resolve_latest)
    elif [ "${version#v}" = "$version" ]; then
        # Accept 1.2.3 as well as v1.2.3; release tags carry the v.
        version="v${version}"
    fi

    local asset="${CLI_FILENAME}_${version}_${TARGET_OS}_${TARGET_ARCH}.tar.gz"
    local url="https://github.com/${GITHUB_REPO}/releases/download/${version}/${asset}"

    echo "Installing ${CLI_FILENAME} ${version} (${TARGET_OS}/${TARGET_ARCH})"
    echo "Downloading ${url}"
    TMP_ROOT=$(mktemp -d)
    download "$url" "${TMP_ROOT}/${asset}" ||
        die "failed to download ${url} (does ${version} exist?)"

    tar -xzf "${TMP_ROOT}/${asset}" -C "$TMP_ROOT"
    # The archive holds one directory named after the asset, with the binary inside.
    local unpacked="${TMP_ROOT}/${asset%.tar.gz}/${CLI_FILENAME}"
    [ -f "$unpacked" ] || die "unexpected archive layout: ${asset} does not contain ${asset%.tar.gz}/${CLI_FILENAME}"

    install_binary "$unpacked"
    note_path
    "${INSTALL_DIR}/${CLI_FILENAME}" version ||
        die "the installed binary failed to run"
}

main "$@"
