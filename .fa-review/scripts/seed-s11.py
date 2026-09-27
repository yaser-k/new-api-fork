"""Session 11 demo data on a fresh scratch instance.

Usage: python3 seed-s11.py <base-url> <sqlite-db-path>
Creates the local admin through /api/setup, a subscription plan (for the
wallet plan card and the purchase dialog) and one consume log that was
retried on three channels (for the admin retry chain), then saves Persian
as the admin's interface language.
"""
import json
import sqlite3
import sys
import time
import urllib.request

BASE = sys.argv[1]
DB = sys.argv[2]
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


call('GET', '/api/setup')
call('POST', '/api/setup', {
    'username': 'admin', 'password': PASSWORD, 'confirmPassword': PASSWORD,
    'SelfUseModeEnabled': False, 'DemoSiteEnabled': False,
})
login = call('POST', '/api/user/login', {'username': 'admin', 'password': PASSWORD})
token = login.get('data', {}).get('access_token')
auth = {'Authorization': f'Bearer {token}', 'New-Api-User': '1'}

for name in ('demo-a', 'demo-b', 'demo-c'):
    call('POST', '/api/channel/', {'mode': 'single', 'channel': {
        'name': name, 'type': 1, 'key': 'sk-demo-not-real', 'base_url': '',
        'models': 'gpt-4o', 'group': 'default', 'status': 1}}, auth)

call('POST', '/api/option/payment_compliance', {'confirmed': True}, auth)
call('POST', '/api/subscription/admin/plans', {'plan': {
    'title': 'Basic', 'subtitle': 'Light usage', 'price_amount': 9.9,
    'duration_unit': 'month', 'duration_value': 1, 'enabled': True,
    'sort_order': 10, 'total_amount': 5000000}}, auth)

now = int(time.time())
con = sqlite3.connect(DB)
other = json.dumps({'model_ratio': 1.25, 'completion_ratio': 4,
                    'group_ratio': 1, 'frt': 850,
                    'admin_info': {'use_channel': ['1', '2', '3']}})
con.execute(
    'INSERT INTO logs (user_id, created_at, type, content, username, '
    'token_name, model_name, quota, prompt_tokens, completion_tokens, '
    'use_time, is_stream, channel_id, token_id, "group", ip, other) '
    'VALUES (1, ?, 2, "", "admin", "demo-key", "gpt-4o", 4210, 1250, 380, '
    '3, 1, 3, 1, "default", "203.0.113.7", ?)',
    (now - 600, other))
con.commit()
con.close()

if len(sys.argv) > 3:
    call('PUT', '/api/user/self', {'language': sys.argv[3]}, auth)
print('SEED_DONE')
