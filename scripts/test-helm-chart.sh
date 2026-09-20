#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHART_PATH="${ROOT_DIR}/charts/observer"
CHART_DEPS_PATH="${CHART_PATH}/charts"
CHART_INPUT="${1:-${CHART_PATH}}"

assert_render_fails() {
  local name="$1"
  local expected="$2"
  shift 2

  local output
  if output=$(helm template observer "${CHART_INPUT}" "$@" 2>&1); then
    echo "ERROR: ${name} unexpectedly rendered successfully"
    exit 1
  fi

  if [[ "${output}" != *"${expected}"* ]]; then
    echo "ERROR: ${name} failed without the expected message: ${expected}"
    printf '%s\n' "${output}"
    exit 1
  fi
}

render_matrix() {
  local chart_input="$1"

  echo "==> Helm lint (${chart_input})"
  helm lint "${chart_input}"

  echo "==> Helm template matrix (${chart_input})"
  helm template observer "${chart_input}" > /dev/null
  helm template observer "${chart_input}" --values "${CHART_PATH}/values-aio.yaml" > /dev/null
  helm template observer "${chart_input}" --values "${CHART_PATH}/values-production.yaml" > /dev/null
  helm template observer "${chart_input}" --values "${CHART_PATH}/values-aio-gateway.yaml" > /dev/null
  helm template observer "${chart_input}" \
    --set nats.enabled=false \
    --set externalNats.url=nats://nats.example.com:4222 > /dev/null
  helm template observer "${chart_input}" \
    --set postgresql.enabled=false \
    --set postgres.host=postgres.example.com > /dev/null
  helm template observer "${chart_input}" \
    --set mongodb.enabled=false \
    --set externalDatabase.host=mongo.example.com > /dev/null

  echo "==> Expected external-dependency failures (${chart_input})"
  assert_render_fails "missing external NATS URL" \
    "externalNats/url" \
    --set nats.enabled=false
  assert_render_fails "missing external PostgreSQL host" \
    "postgres/host" \
    --set postgresql.enabled=false
  assert_render_fails "missing external MongoDB host" \
    "externalDatabase/host" \
    --set mongodb.enabled=false
}

if [[ "${CHART_INPUT}" == *.tgz ]]; then
  echo "==> Inspect packaged chart"
  tar -tzf "${CHART_INPUT}" | grep -q '/Chart.yaml$'
  tar -tzf "${CHART_INPUT}" | grep -q '/charts/postgresql-'
  tar -tzf "${CHART_INPUT}" | grep -q '/charts/mongodb-'
  tar -tzf "${CHART_INPUT}" | grep -q '/charts/nats-'
  render_matrix "${CHART_INPUT}"
  echo "Packaged Helm chart tests completed."
  exit 0
fi

render_matrix "${CHART_INPUT}"

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

package_dir="$(mktemp -d)"
trap 'rm -rf "${package_dir}"' EXIT

echo "==> Package chart for artifact validation"
helm package "${CHART_PATH}" --destination "${package_dir}" > /dev/null
package_path="${package_dir}/$(basename "${CHART_PATH}")-$(helm show chart "${CHART_PATH}" | awk '/^version:/ { print $2 }').tgz"
render_matrix "${package_path}"

echo "Helm chart source and packaged-artifact tests completed."
