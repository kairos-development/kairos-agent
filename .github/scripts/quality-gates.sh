#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

failures=0

report_failure() {
  local title="$1"
  local body="$2"

  echo "::error title=${title}::${title}"
  printf '%s\n' "$body"
  failures=$((failures + 1))
}

check_no_ignored_decimal_parse() {
  local matches
  matches="$(
    grep -RInE '([[:alnum:]_.]+|_)[[:space:]]*,[[:space:]]*_[[:space:]]*(:=|=)[[:space:]]*decimal\.NewFromString|_[[:space:]]*(:=|=)[[:space:]]*decimal\.NewFromString' \
      internal/ \
      --include='*.go' \
      --exclude='*_test.go' \
      || true
  )"

  if [[ -n "$matches" ]]; then
    report_failure \
      "ignored decimal parsing in production path" \
      "Do not ignore decimal.NewFromString errors in trading/storage paths. Use explicit parsing helpers and return contextual errors.

$matches"
  fi
}

check_no_float64_trading_path() {
  local matches
  matches="$(
    grep -RIn 'float64' \
      internal/domain/ \
      internal/service/ \
      internal/providers/bybit/ \
      internal/providers/storage/ \
      --include='*.go' \
      --exclude='*_test.go' \
      | grep -v 'avoid float64' \
      || true
  )"

  if [[ -n "$matches" ]]; then
    report_failure \
      "float64 in trading path" \
      "Trading path must use shopspring/decimal for money, price, quantity, fee, margin and PnL values.

$matches"
  fi
}

check_no_decimal_from_float_trading_path() {
  local matches
  matches="$(
    grep -RIn 'decimal\.NewFromFloat' \
      internal/domain/ \
      internal/service/ \
      internal/providers/bybit/ \
      internal/providers/storage/ \
      internal/strategy/ \
      --include='*.go' \
      --exclude='*_test.go' \
      || true
  )"

  if [[ -n "$matches" ]]; then
    report_failure \
      "decimal.NewFromFloat in trading path" \
      "Trading constants and values must be exact decimals. Use NewFromInt/Shift or checked string parsing instead of decimal.NewFromFloat.

$matches"
  fi
}

check_tui_layering() {
  local matches
  matches="$(
    grep -RInE 'github\.com/kairos-development/kairos-agent/internal/(engine|providers|storage|runtime|domain/connector)' \
      internal/tui/ \
      --include='*.go' \
      --exclude='*_test.go' \
      || true
  )"

  if [[ -n "$matches" ]]; then
    report_failure \
      "TUI dependency boundary violation" \
      "TUI components/views must use QueryService, ActionService and viewmodel boundaries only. Do not import engine/providers/storage/runtime/connector directly.

$matches"
  fi
}

check_no_prod_panics() {
  local matches
  matches="$(
    grep -RInE 'panic\(' \
      internal/ cmd/ \
      --include='*.go' \
      --exclude='*_test.go' \
      || true
  )"

  if [[ -n "$matches" ]]; then
    report_failure \
      "panic in production path" \
      "Production code should return typed/wrapped errors or use SafeGo recovery boundaries. Panic is reserved for tests or extremely narrow invariants with approval.

$matches"
  fi
}

check_no_secret_logging_keywords() {
  local matches
  matches="$(
    grep -RInE 'WithField|WithFields' \
      internal/ cmd/ \
      --include='*.go' \
      --exclude='*_test.go' \
      | grep -Ei '(api_secret|secret|password|token|private_key)' \
      || true
  )"

  if [[ -n "$matches" ]]; then
    report_failure \
      "possible secret logging" \
      "Secrets must be redacted before logging. Review these log fields manually or replace them with masked values.

$matches"
  fi
}

check_no_ignored_decimal_parse
check_no_float64_trading_path
check_no_decimal_from_float_trading_path
check_tui_layering
check_no_prod_panics
check_no_secret_logging_keywords

if [[ "$failures" -gt 0 ]]; then
  echo "Quality gates failed: $failures"
  exit 1
fi

echo "Quality gates passed"
