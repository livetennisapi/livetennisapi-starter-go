#!/bin/sh
# Truth-pin: fail CI when stale product facts reappear in tracked text files.
# POSIX sh, no dependencies beyond git + grep.
set -u
cd "$(dirname "$0")/.." || exit 1

# Tracked text files, minus this script (it names the forbidden patterns),
# CHANGELOG history entries, locks, and binary assets.
files=$(git ls-files | grep -viE \
  '(^|/)scripts/truthcheck\.sh$|(^|/)changelog\.md$|(^|/)(package-lock\.json|go\.sum)$|\.(png|jpe?g|gif|svg|ico|webp|woff2?)$')
[ -n "$files" ] || exit 0

status=0

forbid() { # forbid <label> <regex>
  hits=$(printf '%s\n' "$files" | xargs grep -inE "$2" 2>/dev/null)
  if [ -n "$hits" ]; then
    echo "truthcheck: FORBIDDEN ($1):"
    printf '%s\n' "$hits"
    status=1
  fi
}

# The FREE tier is 100/day (quota grid of 2026-08-06) — older figures are wrong.
forbid "stale 100k daily quota"        '(100,000|100k)( requests| calls)?( ?/ ?| per )day'
forbid "free tier paired with 1k/day"  '[Ff]ree[^.]{0,60}(1,000|1k)( requests| calls)?( ?/ ?| per )day'
# Docs live at docs.livetennisapi.com, not under the marketing site.
forbid "wrong docs URL"                'livetennisapi\.com/docs'
# Org identity only — no personal handle in anything machine-read.
forbid "personal handle"               'bensynapse'
# The daily reset is a local-midnight-derived instant, never midnight UTC.
forbid "midnight UTC claim"            'midnight UTC'

# If quotas are stated at all, the FREE figure and the docs host must be present.
if printf '%s\n' "$files" | xargs grep -qiE '[0-9][0-9,k]* ?(requests|calls)? ?/ ?day' 2>/dev/null; then
  if ! printf '%s\n' "$files" | xargs grep -qE '100( requests)? ?/ ?day' 2>/dev/null; then
    echo "truthcheck: quotas are stated but the FREE figure '100/day' is missing"
    status=1
  fi
  if ! printf '%s\n' "$files" | xargs grep -q 'docs\.livetennisapi\.com' 2>/dev/null; then
    echo "truthcheck: quotas are stated but docs.livetennisapi.com is never referenced"
    status=1
  fi
fi

[ "$status" -eq 0 ] && echo "truthcheck: OK"
exit "$status"
