#!/usr/bin/env python3
"""Exercise the real installer/fetcher with local, synthetic release archives."""
import hashlib
import io
import os
from pathlib import Path
import platform
import subprocess
import tarfile
import tempfile

ROOT = Path(__file__).resolve().parents[1]
OS = {'Linux': 'linux', 'Darwin': 'darwin'}[platform.system()]
ARCH = {'x86_64': 'amd64', 'arm64': 'arm64', 'aarch64': 'arm64'}[platform.machine()]
TARGET = f'{OS}_{ARCH}'

def archive(directory, entries):
    path = directory / f'{TARGET}.tar.gz'
    with tarfile.open(path, 'w:gz') as tar:
        for name, data in entries.items():
            header = tarfile.TarInfo(name)
            header.size, header.mode = len(data), 0o755
            tar.addfile(header, io.BytesIO(data))
    path.with_name(path.name + '.sha256').write_text(hashlib.sha256(path.read_bytes()).hexdigest())
    return path

def run(script, env, *args):
    return subprocess.run(['sh', str(ROOT / script), *map(str, args)], env={**os.environ, **env}, capture_output=True, text=True)

with tempfile.TemporaryDirectory() as temp:
    base = Path(temp)
    release = base / 'release'; release.mkdir()
    prefix = base / 'bin with spaces'
    body = b'#!/bin/sh\nprintf "fixture\\n"\n'
    bundle = archive(release, {'crontab-dashboard':body, 'cronitor':body})
    env = {'DASHBOARD_VERSION':'v0.1.0', 'DASHBOARD_RELEASE_BASE_URL':release.as_uri(), 'INSTALL_DIR':str(prefix)}
    result = run('install.sh', env)
    assert result.returncode == 0, result.stderr
    assert (prefix/'crontab-dashboard').is_symlink()
    assert (prefix/'cronitor').is_symlink()
    assert (prefix/'crontab-dashboard').resolve().parent == (prefix/'cronitor').resolve().parent
    installed = (prefix/'crontab-dashboard').resolve()
    companion = (prefix/'cronitor').resolve()
    assert installed.read_bytes() == body and companion.read_bytes() == body
    # Failed checksum must leave the previous install and CLI usable.
    bundle.with_name(bundle.name+'.sha256').write_text('0'*64)
    result = run('install.sh', env)
    assert result.returncode != 0 and 'checksum mismatch' in result.stderr
    assert (prefix/'crontab-dashboard').resolve() == installed
    assert (prefix/'cronitor').resolve() == companion
    archive(release, {'crontab-dashboard':body, 'cronitor':body})
    result = run('install.sh', {**env, 'DASHBOARD_VERSION':'v0.1.1'})
    assert result.returncode == 0, result.stderr
    installed = (prefix/'crontab-dashboard').resolve()
    assert installed.parent == (prefix/'cronitor').resolve().parent
    # Existing CLI belongs to its operator and must not be replaced.
    existing = base/'existing'; existing.mkdir()
    (existing/'cronitor').write_bytes(b'original-cli')
    archive(release, {'crontab-dashboard':body, 'cronitor':body})
    result = run('install.sh', {**env, 'INSTALL_DIR':str(existing)})
    assert result.returncode == 0, result.stderr
    assert (existing/'cronitor').read_bytes() == b'original-cli'
    assert ((existing/'crontab-dashboard').resolve().parent/'cronitor').read_bytes() == body
    # Missing companion cannot publish a partial dashboard installation.
    archive(release, {'crontab-dashboard':body})
    result = run('install.sh', env)
    assert result.returncode != 0
    assert (prefix/'crontab-dashboard').resolve() == installed
    cli_release = base/'cli'; cli_release.mkdir()
    cli = archive(cli_release, {'cronitor':body})
    dest = base/'fetched-cronitor'
    env = {'CLI_RELEASE_BASE_URL':cli_release.as_uri()}
    result = run('scripts/fetch-cli.sh', env, TARGET, dest)
    assert result.returncode == 0 and dest.read_bytes() == body, result.stderr
    cli.with_name(cli.name+'.sha256').write_text('0'*64)
    result = run('scripts/fetch-cli.sh', env, TARGET, dest)
    assert result.returncode != 0 and dest.read_bytes() == body
print('Installer/fetcher checks passed: complete bundle, checksum failures, existing CLI, missing companion.')
