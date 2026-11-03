#!/usr/bin/env bash
# Builds libheif + libde265 as real WebAssembly (not wasm2js) for the HEIC
# fallback decoder in web/src/interop/codecs.ts. Run inside an Emscripten
# toolchain, e.g. the pinned emscripten/emsdk image used by
# .github/workflows/libheif-wasm.yml:
#
#   docker run --rm -v "$PWD:/src" -w /src emscripten/emsdk:3.1.61@sha256:30aa9fed39cc7810cc4338c77828bd513a6ee6b60c6b141bc6d9760aea5791e3 \
#     scripts/libheif-wasm/build.sh out/libheif
#
# Outputs libheif.js, libheif.wasm and SHA256SUMS into the given directory.
set -euo pipefail

LIBHEIF_VERSION=1.23.7
LIBHEIF_SHA256=f7483e818503ea6224c8863e4a56ac02c7089021a98472c811c12aa19faf2475
LIBDE265_VERSION=1.1.3
LIBDE265_SHA256=554228bd17788c99a7e63b37ab5634722190e6e2bf60c1dcb01cef328e133905

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT="$(mkdir -p "${1:-out}" && cd "${1:-out}" && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

fetch() { # url file sha256
  curl -fsSL --retry 3 -o "$WORK/$2" "$1"
  echo "$3  $WORK/$2" | sha256sum -c -
}

fetch "https://github.com/strukturag/libheif/releases/download/v${LIBHEIF_VERSION}/libheif-${LIBHEIF_VERSION}.tar.gz" \
  "libheif-${LIBHEIF_VERSION}.tar.gz" "$LIBHEIF_SHA256"
fetch "https://github.com/strukturag/libde265/releases/download/v${LIBDE265_VERSION}/libde265-${LIBDE265_VERSION}.tar.gz" \
  "libde265-${LIBDE265_VERSION}.tar.gz" "$LIBDE265_SHA256"

tar -xzf "$WORK/libheif-${LIBHEIF_VERSION}.tar.gz" -C "$WORK"
SRC="$WORK/libheif-${LIBHEIF_VERSION}"
patch -d "$SRC" -p1 < "$HERE/build-emscripten.patch"

mkdir -p "$SRC/buildjs"
# build-emscripten.sh reuses an already-downloaded tarball in its working directory.
mv "$WORK/libde265-${LIBDE265_VERSION}.tar.gz" "$SRC/buildjs/"
cd "$SRC/buildjs"
# libheif collects exports from libheif.a only, so libde265's version getter
# (the runtime security gate) and UTF8ToString are exported explicitly.
USE_WASM=1 USE_UNSAFE_EVAL=0 USE_ES6=1 USE_TYPESCRIPT=0 \
LIBDE265_VERSION="$LIBDE265_VERSION" \
EXTRA_EXPORTED_FUNCTIONS=",_de265_get_version" \
EXTRA_EMCC_FLAGS="-sEXPORTED_RUNTIME_METHODS=UTF8ToString" \
  ../build-emscripten.sh ..

cp libheif.js libheif.wasm "$OUT/"
cd "$OUT"
sha256sum libheif.js libheif.wasm > SHA256SUMS
{
  echo "libheif ${LIBHEIF_VERSION} (sha256 ${LIBHEIF_SHA256})"
  echo "libde265 ${LIBDE265_VERSION} (sha256 ${LIBDE265_SHA256})"
  echo "emcc $(emcc --version | head -n1)"
} > BUILDINFO
cat BUILDINFO SHA256SUMS
ls -l libheif.js libheif.wasm
