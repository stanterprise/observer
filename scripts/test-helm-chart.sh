#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHART_PATH="${ROOT_DIR}/charts/observer"
CHART_DEPS_PATH="${CHART_PATH}/charts"

echo "==> Helm lint"
helm lint "${CHART_PATH}"

if [ ! -d "${CHART_DEPS_PATH}" ] || [ -z "$(find "${CHART_DEPS_PATH}" -mindepth 1 -maxdepth 1 2>/dev/null)" ]; then
  echo "==> Helm dependencies are missing; attempting to build chart dependencies"
  if ! helm dependency build "${CHART_PATH}"; then
    if [ "${HELM_REQUIRE_DEPENDENCIES:-false}" = "true" ]; then
      echo "ERROR: failed to fetch Helm dependencies and HELM_REQUIRE_DEPENDENCIES=true"
      exit 1
    fi
    echo "WARNING: skipping template rendering because dependencies could not be fetched"
    echo "         Re-run with network access or set HELM_REQUIRE_DEPENDENCIES=true to fail hard."
    exit 0
  fi
fi

echo "==> Helm template (default values)"
helm template observer "${CHART_PATH}" > /dev/null

echo "==> Helm template (AIO values)"
helm template observer "${CHART_PATH}" --values "${CHART_PATH}/values-aio.yaml" > /dev/null

echo "==> Helm template (production values)"
helm template observer "${CHART_PATH}" --values "${CHART_PATH}/values-production.yaml" > /dev/null

echo "Helm chart tests completed."
