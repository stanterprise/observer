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

  if [[ "${output}" != *"${expected}"* && "${output//\//.}" != *"${expected}"* ]]; then
    echo "ERROR: ${name} failed without the expected message: ${expected}"
    printf '%s\n' "${output}"
    exit 1
  fi
}

render_matrix() {
  local chart_input="$1"
  local CHART_INPUT="$chart_input"
  local expected_image_tag
  local rendered_images
  local external_connection_output

  expected_image_tag="$(helm show chart "${chart_input}" | awk '$1 == "appVersion:" { gsub(/"/, "", $2); print $2 }')"
  rendered_images="$(helm template observer "${chart_input}" | sed -n 's/.*image: ghcr.io\/stanterprise\/observer\/[^:]*:\([^" ]*\).*/\1/p' | sort -u)"
  if [[ -z "${expected_image_tag}" || "${rendered_images}" != "${expected_image_tag}" ]]; then
    echo "ERROR: rendered Observer image tags do not match appVersion"
    echo "Expected: ${expected_image_tag}"
    echo "Rendered: ${rendered_images}"
    exit 1
  fi

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
  external_connection_output="$(helm template observer "${chart_input}" \
    --set postgresql.enabled=false \
    --set postgres.host=postgres.example.com \
    --set-string 'postgres.password=p@:/#%' \
    --set mongodb.enabled=false \
    --set externalDatabase.host=mongo.example.com \
    --set-string 'externalDatabase.password=p@:/#%')"
  if [[ "${external_connection_output}" != *'POSTGRES_DSN: "postgres://observer:p%40%3A%2F%23%25@postgres.example.com:5432/observer?sslmode=disable"'* ]]; then
    echo "ERROR: generated PostgreSQL DSN is malformed or does not escape credentials"
    exit 1
  fi
  if [[ "${external_connection_output}" != *'MONGODB_URI: "mongodb://observer:p%40%3A%2F%23%25@mongo.example.com:27017/observer?authSource=admin"'* ]]; then
    echo "ERROR: generated MongoDB URI is malformed or does not escape credentials"
    exit 1
  fi
  helm template observer "${chart_input}" \
    --set extraEnvFrom[0].configMapRef.name=chart-test-env \
    --set storage.s3.existingSecret=chart-test-storage > /dev/null
  helm template observer "${chart_input}" \
    --set postgresql.auth.existingSecret=chart-test-postgres \
    --set postgresql.auth.secretKeys.userPasswordKey=custom-password \
    --set mongodb.auth.existingSecret=chart-test-mongodb \
    --set mongodb.auth.rootUser=chart-test-root > /dev/null
  helm template observer "${chart_input}" \
    --set-json 'distributed.web.env.API_BACKEND_PORT=null' > /dev/null

  echo "==> Expected external-dependency failures (${chart_input})"
  assert_render_fails "missing external NATS URL" \
    "externalNats.url" \
    --set nats.enabled=false
  assert_render_fails "missing external PostgreSQL host" \
    "postgres.host" \
    --set postgresql.enabled=false
  assert_render_fails "missing external MongoDB host" \
    "externalDatabase.host" \
    --set mongodb.enabled=false
  assert_render_fails "managed connection variables in extraEnv" \
    "extraEnv.NATS_URL" \
    --set-string extraEnv.NATS_URL=nats://not-allowed.example.com
}

if [[ "${CHART_INPUT}" == *.tgz ]]; then
  echo "==> Inspect packaged chart"
  package_contents="$(tar -tzf "${CHART_INPUT}")"
  grep -Fxq 'observer/Chart.yaml' <<< "${package_contents}"
  grep -Fxq 'observer/charts/postgresql/Chart.yaml' <<< "${package_contents}"
  grep -Fxq 'observer/charts/mongodb/Chart.yaml' <<< "${package_contents}"
  grep -Fxq 'observer/charts/nats/Chart.yaml' <<< "${package_contents}"
  render_matrix "${CHART_INPUT}"
  echo "Packaged Helm chart tests completed."
  exit 0
fi

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

render_matrix "${CHART_INPUT}"

package_dir="$(mktemp -d)"
trap 'rm -rf "${package_dir}"' EXIT

echo "==> Package chart for artifact validation"
helm package "${CHART_PATH}" --destination "${package_dir}" > /dev/null
package_path="${package_dir}/$(basename "${CHART_PATH}")-$(helm show chart "${CHART_PATH}" | awk '/^version:/ { print $2 }').tgz"
render_matrix "${package_path}"

echo "Helm chart source and packaged-artifact tests completed."
