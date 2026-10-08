#!/usr/bin/env python3
"""Process supervisor checks, including failure and signal cleanup."""
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]
SHELL = shutil.which('sh')
assert SHELL
with tempfile.TemporaryDirectory() as temp:
    directory=Path(temp)
    for name in ('crond','crontab-dashboard'):
        (directory/name).write_text(f'''#!/bin/sh
trap 'echo stopped >> "{directory/name}.stopped"; exit 0' TERM INT
if [ "$FAIL_PROCESS" = "{name}" ]; then exit 23; fi
while :; do sleep 0.1; done
''')
        (directory/name).chmod(0o755)
    env={**os.environ,'PATH':str(directory)+':'+os.environ['PATH'],'CRONITOR_DASH_USER':'fixture','CRONITOR_DASH_PASS':'synthetic-not-real','FAIL_PROCESS':''}
    process=subprocess.Popen([SHELL,str(ROOT/'docker-entrypoint.sh')],env=env)
    time.sleep(0.3); process.terminate()
    assert process.wait(timeout=5)==0
    assert all((directory/(n+'.stopped')).exists() for n in ('crond','crontab-dashboard'))
    for failing in ('crond','crontab-dashboard'):
        process=subprocess.Popen([SHELL,str(ROOT/'docker-entrypoint.sh')],env={**env,'FAIL_PROCESS':failing})
        assert process.wait(timeout=5)==23
    result=subprocess.run([SHELL,str(ROOT/'docker-entrypoint.sh')],env={**env,'CRONITOR_DASH_PASS':''},capture_output=True)
    assert result.returncode!=0
print('Container supervisor checks passed: child failures, shutdown, credentials.')
