#!/usr/bin/env bash
set -euo pipefail

# Infer current lifecycle phase for a feature by checking doc state.
# Usage: check-status.sh <feature-name>

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <feature-name>"
  exit 1
fi

FEATURE="$1"
DOCS="docs/ai"

exists()      { [[ -f "$1" ]]; }
# A doc is "filled" when it contains at least one authored section heading (## ...)
# that is NOT part of a bare template (templates typically have no filled content).
has_content() { [[ -f "$1" ]] && grep -qE '^## ' "$1" 2>/dev/null; }

REQ="$DOCS/${FEATURE}/requirements.md"
DES="$DOCS/${FEATURE}/design.md"
PLN="$DOCS/${FEATURE}/planning.md"
IMP="$DOCS/${FEATURE}/implementation.md"
TST="$DOCS/${FEATURE}/testing.md"

echo "=== Status: $FEATURE ==="

# Show FILLED / EMPTY (exists but no content) / MISS for each doc
for doc in "$REQ" "$DES" "$PLN" "$IMP" "$TST"; do
  if has_content "$doc"; then
    echo "[FILLED] $doc"
  elif exists "$doc"; then
    echo "[EMPTY]  $doc"
  else
    echo "[MISS]   $doc"
  fi
done

TOTAL=0; DONE=0; TODO=0
if exists "$PLN"; then
  TOTAL=$(grep -c '^\s*- \[' "$PLN" 2>/dev/null || true)
  DONE=$(grep -c '^\s*- \[x\]' "$PLN" 2>/dev/null || true)
  TOTAL=${TOTAL:-0}
  DONE=${DONE:-0}
  TODO=$((TOTAL - DONE))
  echo ""
  echo "Planning: $DONE/$TOTAL tasks done, $TODO remaining"
fi

echo ""
echo "--- Suggested phase ---"
if ! has_content "$REQ"; then
  echo "Phase 1 (New Requirement) — requirements doc missing or empty"
elif ! has_content "$DES"; then
  echo "Phase 2 (Review Requirements) — requirements filled, design doc missing or empty"
elif ! has_content "$PLN"; then
  echo "Phase 3 (Review Design) — design filled, planning doc missing or empty"
elif [[ $TOTAL -eq 0 ]]; then
  echo "Phase 3→4 (Review Design → Execute Plan) — planning doc has no tasks yet"
elif [[ $TODO -gt 0 ]]; then
  echo "Phase 4 (Execute Plan) — $TODO/$TOTAL tasks remaining"
elif ! has_content "$TST"; then
  echo "Phase 6 (Check Implementation) — all tasks done, verify against design before writing tests"
elif has_content "$TST" && ! has_content "$IMP"; then
  echo "Phase 7 (Write Tests) — tests started, implementation notes doc still empty"
else
  echo "Phase 8 (Code Review) — all phases complete, ready for final review"
fi
