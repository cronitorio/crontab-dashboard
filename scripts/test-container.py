#!/usr/bin/env python3
"""Smoke-test the built image, including a real cron invocation and persistence."""
import base64
import json
import os
import subprocess
import time
import urllib.error
import urllib.request
import uuid

IMAGE=os.environ.get('DASHBOARD_TEST_IMAGE','crontab-dashboard:test')
name='dashboard-smoke-'+uuid.uuid4().hex[:10]
config=name+'-config'; cron=name+'-cron'

def docker(*args,check=True):
    return subprocess.run(['docker',*args],capture_output=True,text=True,check=check).stdout.strip()

try:
    docker('run','-d','--name',name,'-e','CRONITOR_DASH_USER=fixture','-e','CRONITOR_DASH_PASS=synthetic-not-real','-p','127.0.0.1::9000','-v',config+':/etc/cronitor','-v',cron+':/var/spool/cron/crontabs',IMAGE)
    port=docker('port',name,'9000/tcp').rsplit(':',1)[1]
    url='http://127.0.0.1:'+port
    deadline=time.monotonic()+30
    while True:
        try:
            urllib.request.urlopen(url,timeout=2)
            raise AssertionError('Unauthenticated dashboard allowed')
        except urllib.error.HTTPError as err:
            assert err.code==401; break
        except (OSError,urllib.error.URLError):
            if time.monotonic()>deadline: raise
            time.sleep(.2)
    auth='Basic '+base64.b64encode(b'fixture:synthetic-not-real').decode()
    request=urllib.request.Request(url,headers={'Authorization':auth})
    with urllib.request.urlopen(request,timeout=5) as response:
        assert response.status==200 and b'<html' in response.read().lower()
    processes=docker('exec',name,'ps')
    assert 'crond' in processes and 'crontab-dashboard' in processes
    # Use the dashboard API to persist a real user crontab, then let cron run it.
    request=urllib.request.Request(url+'/api/crontabs',headers={'Authorization':auth})
    with urllib.request.urlopen(request,timeout=5) as response:
        token=response.headers['X-CSRF-Token']
    def api(path, payload):
        global token
        request=urllib.request.Request(url+path,data=json.dumps(payload).encode(),headers={'Authorization':auth,'X-CSRF-Token':token,'Content-Type':'application/json'})
        with urllib.request.urlopen(request,timeout=10) as response:
            token=response.headers.get("X-CSRF-Token",token)
            return response.read()
    api('/api/crontabs', {'filename':'user:root'})
    api('/api/jobs',{'name':'Container smoke','command':'printf tick >> /tmp/cron-smoke','expression':'* * * * *','run_as_user':'root','crontab_filename':'user:root','monitored':False})
    assert 'cron-smoke' in docker('exec',name,'crontab','-l')
    output=api('/api/jobs/run',{'command':'printf manual-smoke','with_monitoring':False})
    assert b'manual-smoke' in output and b'Exit code 0' in output
    api('/api/settings',{'CRONITOR_DASH_USER':'fixture','CRONITOR_DASH_PASS':'synthetic-not-real','CRONITOR_HOSTNAME':'container-smoke'})
    print('Waiting for the container cron daemon to execute its smoke job...',flush=True)
    deadline=time.monotonic()+85
    while True:
        if docker('exec',name,'sh','-c','test -s /tmp/cron-smoke && echo ready',check=False)=='ready': break
        if time.monotonic()>deadline: raise AssertionError('cron did not execute the scheduled job')
        time.sleep(2)
    docker('stop','-t','10',name)
    assert docker('inspect','--format','{{.State.ExitCode}}',name)=='0'
    docker('rm',name)
    docker('run','-d','--name',name,'-e','CRONITOR_DASH_USER=fixture','-e','CRONITOR_DASH_PASS=synthetic-not-real','-v',config+':/etc/cronitor','-v',cron+':/var/spool/cron/crontabs',IMAGE)
    assert 'cron-smoke' in docker('exec',name,'crontab','-l')
    assert 'container-smoke' in docker('exec',name,'cat','/etc/cronitor/cronitor.json')
    print('Container smoke checks passed: auth, both daemons, real cron job, persistence, shutdown.')
except Exception:
    print(docker('logs',name,check=False))
    raise
finally:
    docker('rm','-f',name,check=False)
    docker('volume','rm',config,cron,check=False)
