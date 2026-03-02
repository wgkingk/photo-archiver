#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MAC_APP_DIR="${ROOT_DIR}/mac-app"
BUILD_DIR="${MAC_APP_DIR}/build"
BACKEND_DIR="${BUILD_DIR}/backend"
DERIVED_DATA_DIR="${BUILD_DIR}/DerivedData"
DIST_DIR="${ROOT_DIR}/dist"

echo "==> Building backend service (universal)"
mkdir -p "${BACKEND_DIR}"

CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o "${BACKEND_DIR}/photo-archiver-service-arm64" "${ROOT_DIR}/cmd/service"
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o "${BACKEND_DIR}/photo-archiver-service-amd64" "${ROOT_DIR}/cmd/service"
lipo -create -output "${BACKEND_DIR}/photo-archiver-service" "${BACKEND_DIR}/photo-archiver-service-arm64" "${BACKEND_DIR}/photo-archiver-service-amd64"
chmod +x "${BACKEND_DIR}/photo-archiver-service"

echo "==> Generating Xcode project"
cd "${MAC_APP_DIR}"
xcodegen generate

echo "==> Building release app"
xcodebuild \
  -project "${MAC_APP_DIR}/PhotoArchiverMac.xcodeproj" \
  -scheme "PhotoArchiverMac" \
  -configuration Release \
  -destination "platform=macOS" \
  -derivedDataPath "${DERIVED_DATA_DIR}" \
  build

APP_PATH="${DERIVED_DATA_DIR}/Build/Products/Release/PhotoArchiverMac.app"
if [ ! -d "${APP_PATH}" ]; then
  echo "Build finished but app not found: ${APP_PATH}"
  exit 1
fi

echo "==> Packaging app"
mkdir -p "${DIST_DIR}"
rm -rf "${DIST_DIR}/PhotoArchiverMac.app"
cp -R "${APP_PATH}" "${DIST_DIR}/PhotoArchiverMac.app"

echo "Done. App output: ${DIST_DIR}/PhotoArchiverMac.app"
