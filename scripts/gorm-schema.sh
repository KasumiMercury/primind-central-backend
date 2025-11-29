#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

dialect="postgres"
packages=(
  "./internal/auth/infra/repository"
  "./internal/task/infra/repository"
)

for pkg in "${packages[@]}"; do
  go run -mod=mod ariga.io/atlas-provider-gorm \
    load \
    --path "${pkg}" \
    --dialect "${dialect}"
done
