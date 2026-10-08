#!/bin/sh
set -eu
if [ -z "${CRONITOR_DASH_USER:-}" ] || [ -z "${CRONITOR_DASH_PASS:-}" ]; then
 echo 'Set CRONITOR_DASH_USER and CRONITOR_DASH_PASS at runtime.' >&2
 exit 1
fi
crond -f -l 8 &
cron_pid=$!
crontab-dashboard "$@" &
dash_pid=$!
stop() {
 trap - TERM INT
 kill -TERM "$dash_pid" "$cron_pid" 2>/dev/null || true
 wait "$dash_pid" 2>/dev/null || true
 wait "$cron_pid" 2>/dev/null || true
}
trap 'stop; exit 0' TERM INT
status=0
while kill -0 "$cron_pid" 2>/dev/null && kill -0 "$dash_pid" 2>/dev/null; do
 sleep 1
done
if ! kill -0 "$cron_pid" 2>/dev/null; then
 wait "$cron_pid" || status=$?
else
 wait "$dash_pid" || status=$?
fi
stop
exit "$status"
