"""This session's additions to the demo data (after seed.py, seed6.py, seed7.py, seed8.py, seed10.py).

Usage: python3 seed-final.py <base-url>

Promotes and demotes demo-user (user.manage audit entries with an action)
and overrides its quota (a quota audit summary with a before and after
value). Everything goes through the admin API of the local demo server.
"""
import json
import sys
import urllib.request

BASE = sys.argv[1]
PASSWORD = 'DemoPass-2026!'


def call(method, path, body=None, headers=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(BASE + path, data=data, method=method)
    req.add_header('Content-Type', 'application/json')
    for k, v in (headers or {}).items():
        req.add_header(k, v)
    try:
        with urllib.request.urlopen(req) as resp:
            text = resp.read().decode()
    except urllib.error.HTTPError as err:
        text = err.read().decode()
    print(f'{method} {path} -> {text[:160]}')
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        return {}


login = call('POST', '/api/user/login', {'username': 'admin', 'password': PASSWORD})
token = login.get('data', {}).get('access_token')
auth = {'Authorization': f'Bearer {token}', 'New-Api-User': '1'}

call('POST', '/api/user/manage', {'id': 2, 'action': 'promote'}, auth)
call('POST', '/api/user/manage', {'id': 2, 'action': 'demote'}, auth)
call('POST', '/api/user/manage', {'id': 2, 'action': 'add_quota',
                                  'mode': 'override', 'value': 4000000}, auth)
print('SEED_FINAL_DONE')
