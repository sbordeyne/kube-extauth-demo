#!/usr/bin/env bash
#
# Authz scenario runner for api.demo.localhost.
#
# Each scenario is a JSON file with an ordered list of `steps`. A step performs an
# HTTP request, asserts the status code (and optional jq conditions on the response
# body), and may `capture` values into scenario-scoped variables that later steps
# reference with {{var}} templating in `path`, `body`, and `auth`.
#
# Usage:
#   ./run.sh                         # run every testing/*.json
#   ./run.sh path/to/scenario.json   # run a single scenario
#   ./run.sh 'testing/user_admin_*.json'   # run a glob (shell-expanded)
#
# Env:
#   BASE_URL   API base URL (default http://api.demo.localhost)
#
# Requires: curl, jq (jq is pinned in mise.toml).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_URL="${BASE_URL:-http://api.demo.localhost}"

if [ -t 1 ]; then
  GREEN=$'\033[32m'; RED=$'\033[31m'; YELLOW=$'\033[33m'
  DIM=$'\033[2m'; BOLD=$'\033[1m'; RESET=$'\033[0m'
else
  GREEN=; RED=; YELLOW=; DIM=; BOLD=; RESET=
fi

command -v curl >/dev/null 2>&1 || { echo "error: curl not found" >&2; exit 2; }
command -v jq   >/dev/null 2>&1 || { echo "error: jq not found" >&2; exit 2; }

if [ "$#" -gt 0 ]; then
  # Expand each argument as a glob (so both `'user_*.json'` quoted and unquoted
  # work); a pattern that matches nothing is kept literal so it reports as a skip.
  shopt -s nullglob
  FILES=()
  for arg in "$@"; do
    matches=( $arg )
    if [ "${#matches[@]}" -gt 0 ]; then
      FILES+=("${matches[@]}")
    else
      FILES+=("$arg")
    fi
  done
  shopt -u nullglob
else
  FILES=("$SCRIPT_DIR"/*.json)
fi

# Per-run id, used to seed a {{nonce}} variable unique to each scenario so
# resource-creating scenarios (e.g. user creation, which needs a unique email)
# can be re-run without collisions.
RUN_ID="$(date +%s)"
SC_INDEX=0

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
VARS="$TMP/vars"        # per-scenario key<TAB>value store
BODY="$TMP/body"        # last response body
STEP="$TMP/step"        # current step JSON

# Replace every {{key}} in $1 with the captured value from $VARS.
subst() {
  local s="$1" key val
  while IFS=$'\t' read -r key val; do
    [ -n "$key" ] || continue
    s="${s//\{\{$key\}\}/$val}"
  done < "$VARS"
  printf '%s' "$s"
}

body_preview() {
  # First few lines of the response body, indented, for failure output.
  sed 's/^/      /' "$BODY" | head -n 8
}

# ---------------------------------------------------------------------------
# Crafted-JWT support.
#
# A step's `auth` may be a sentinel (@wrongiss / @expired / @wrongkey) instead of
# a captured token. run.sh mints the corresponding RS256 JWT once at startup so
# the negative "bad token is denied" scenarios can run without a live minter.
# Each token carries full admin scopes, so ONLY the crafted flaw (bad issuer /
# past exp / wrong signing key) can cause a denial.
# ---------------------------------------------------------------------------
JWT_KEY="${JWT_KEY:-$SCRIPT_DIR/../sample-api/manifests/config/rs256.key}"
JWT_SCOPES='["post:read","post:create","post:edit","post:delete","post:list","comment:read","comment:create","comment:edit","comment:delete","users:read","users:create","users:edit","users:delete"]'
TOK_WRONGISS=""
TOK_EXPIRED=""
TOK_WRONGKEY=""

b64url() { openssl base64 -e -A | tr '+/' '-_' | tr -d '='; }

# mint_jwt <payload-json> <private-key-file>
mint_jwt() {
  local hdr si sig
  hdr='{"alg":"RS256","typ":"JWT"}'
  si="$(printf '%s' "$hdr" | b64url).$(printf '%s' "$1" | b64url)"
  sig="$(printf '%s' "$si" | openssl dgst -sha256 -sign "$2" -binary | b64url)"
  printf '%s.%s' "$si" "$sig"
}

setup_crafted_tokens() {
  command -v openssl >/dev/null 2>&1 || return 0
  [ -f "$JWT_KEY" ] || {
    echo "${YELLOW}note${RESET}: signing key not found at $JWT_KEY — @wrongiss/@expired/@wrongkey scenarios will send no token" >&2
    return 0
  }
  local now future past sub
  now="$RUN_ID"; future=$((now + 3600)); past=$((now - 3600))
  sub="00000000-0000-0000-0000-0000000000aa"

  TOK_WRONGISS="$(mint_jwt "{\"iss\":\"evil-issuer\",\"sub\":\"$sub\",\"role\":\"admin\",\"scopes\":$JWT_SCOPES,\"iat\":$now,\"exp\":$future}" "$JWT_KEY")"
  TOK_EXPIRED="$(mint_jwt "{\"iss\":\"sample-api\",\"sub\":\"$sub\",\"role\":\"admin\",\"scopes\":$JWT_SCOPES,\"iat\":$past,\"exp\":$past}" "$JWT_KEY")"
  # A throwaway key of the same type, so signature (not claims) is the only flaw.
  openssl genrsa -out "$TMP/wrong.key" 2048 >/dev/null 2>&1
  TOK_WRONGKEY="$(mint_jwt "{\"iss\":\"sample-api\",\"sub\":\"$sub\",\"role\":\"admin\",\"scopes\":$JWT_SCOPES,\"iat\":$now,\"exp\":$future}" "$TMP/wrong.key")"
}

# Run one scenario file. Returns 0 on all-steps-pass, 1 otherwise.
run_scenario() {
  local file="$1"
  : > "$VARS"
  printf 'nonce\t%s\n' "${RUN_ID}-${SC_INDEX}" >> "$VARS"

  local nsteps method path body auth want code
  nsteps="$(jq '.steps | length' "$file")"

  local i=0
  while [ "$i" -lt "$nsteps" ]; do
    jq -c ".steps[$i]" "$file" > "$STEP"

    method="$(jq -r '.request.method' "$STEP")"
    path="$(subst "$(jq -r '.request.path' "$STEP")")"
    auth="$(subst "$(jq -r '.auth // empty' "$STEP")")"
    case "$auth" in
      @wrongiss) auth="$TOK_WRONGISS" ;;
      @expired)  auth="$TOK_EXPIRED" ;;
      @wrongkey) auth="$TOK_WRONGKEY" ;;
    esac
    want="$(jq -r '.expect.status' "$STEP")"

    local -a args
    args=(-sS -X "$method" -H 'Content-Type: application/json')
    [ -n "$auth" ] && args+=(-H "Authorization: Bearer $auth")
    if jq -e '.request | has("body")' "$STEP" >/dev/null; then
      body="$(subst "$(jq -c '.request.body' "$STEP")")"
      args+=(-d "$body")
    fi

    code="$(curl "${args[@]}" -o "$BODY" -w '%{http_code}' "$BASE_URL$path")" || {
      printf '  %sstep %s%s %s %s — %scurl failed%s\n' \
        "$RED" "$i" "$RESET" "$method" "$path" "$RED" "$RESET"
      return 1
    }

    if [ "$code" != "$want" ]; then
      printf '  %sstep %s%s %s %s\n    expected status %s, got %s%s%s\n%s\n' \
        "$RED" "$i" "$RESET" "$method" "$path" \
        "$want" "$RED" "$code" "$RESET" "$(body_preview)"
      return 1
    fi

    # jq body conditions — every expression must be truthy.
    local nq j q
    nq="$(jq '.expect.jq // [] | length' "$STEP")"
    j=0
    while [ "$j" -lt "$nq" ]; do
      q="$(jq -r ".expect.jq[$j]" "$STEP")"
      if ! jq -e "$q" "$BODY" >/dev/null 2>&1; then
        printf '  %sstep %s%s %s %s\n    jq condition failed: %s%s\n%s\n' \
          "$RED" "$i" "$RESET" "$method" "$path" "$q" "$RESET" "$(body_preview)"
        return 1
      fi
      j=$((j+1))
    done

    # captures — write var<TAB>value into the store for later {{var}} use.
    jq -r '.capture // {} | to_entries[] | "\(.key)\t\(.value)"' "$STEP" > "$TMP/caps"
    local ck cexpr cv
    while IFS=$'\t' read -r ck cexpr; do
      [ -n "$ck" ] || continue
      cv="$(jq -r "$cexpr" "$BODY")"
      printf '%s\t%s\n' "$ck" "$cv" >> "$VARS"
    done < "$TMP/caps"

    i=$((i+1))
  done
  return 0
}

setup_crafted_tokens

echo "${BOLD}Running authz scenarios against ${BASE_URL}${RESET}"
echo

pass=0
fail=0
failed=()

for f in "${FILES[@]}"; do
  if [ ! -f "$f" ]; then
    echo "${YELLOW}skip${RESET} (not a file): $f"
    continue
  fi
  name="$(jq -r '.name // empty' "$f" 2>/dev/null || true)"
  [ -n "$name" ] || name="$(basename "$f" .json)"
  desc="$(jq -r '.description // empty' "$f" 2>/dev/null || true)"

  SC_INDEX=$((SC_INDEX+1))
  if run_scenario "$f"; then
    printf '%sPASS%s %s %s%s%s\n' "$GREEN" "$RESET" "$name" "$DIM" "$desc" "$RESET"
    pass=$((pass+1))
  else
    printf '%sFAIL%s %s %s%s%s\n' "$RED" "$RESET" "$name" "$DIM" "$desc" "$RESET"
    fail=$((fail+1))
    failed+=("$name")
  fi
done

echo
if [ "$fail" -eq 0 ]; then
  echo "${GREEN}${BOLD}${pass} passed, 0 failed${RESET}"
else
  echo "${BOLD}${pass} passed, ${RED}${fail} failed${RESET}"
  for n in "${failed[@]}"; do echo "  ${RED}- ${n}${RESET}"; done
  exit 1
fi
