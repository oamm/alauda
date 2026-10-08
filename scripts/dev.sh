#!/usr/bin/env sh
set -eu

db=postgres
reset_password=0
while [ "$#" -gt 0 ]; do
  case "$1" in
    --db|--database)
      [ "$#" -ge 2 ] || { echo "Missing value for $1. Supported values: postgres, sqlite." >&2; exit 2; }
      db=$2
      shift 2
      ;;
    --reset-dev-password)
      reset_password=1
      shift
      ;;
    *) echo "Unknown argument: $1" >&2; exit 2 ;;
  esac
done

case "$db" in
  postgres|sqlite) ;;
  *) printf "Unsupported database provider '%s'. Supported values: postgres, sqlite\n" "$db" >&2; exit 2 ;;
esac

if command -v pwsh >/dev/null 2>&1; then
  if [ "$reset_password" -eq 1 ]; then exec pwsh -NoProfile -File "$(dirname "$0")/run-app.ps1" -DatabaseProvider "$db" -ResetDevPassword; fi
  exec pwsh -NoProfile -File "$(dirname "$0")/run-app.ps1" -DatabaseProvider "$db"
fi
if command -v powershell.exe >/dev/null 2>&1; then
  if [ "$reset_password" -eq 1 ]; then exec powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$(dirname "$0")/run-app.ps1" -DatabaseProvider "$db" -ResetDevPassword; fi
  exec powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$(dirname "$0")/run-app.ps1" -DatabaseProvider "$db"
fi
echo "PowerShell is required to run the existing development bootstrap workflow." >&2
exit 1
