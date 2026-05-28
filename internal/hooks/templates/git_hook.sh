{{MANAGED_START}}
if [ -n "$PREVIOUSLY_ON_SKIP_HOOK" ]; then
  exit 0
fi

if [ ! -t 1 ]; then
  exit 0
fi

repo_root="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0
cd "$repo_root" || exit 0

{{EXECUTABLE}} summary --simple
{{MANAGED_END}}
