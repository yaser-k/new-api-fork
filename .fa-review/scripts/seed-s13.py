"""Session 13 demo data (after the upstream merge), written once and copied
to every instance.

Usage: python3 seed-s13.py <base-url> <sqlite-db-path>
Creates the local admin through /api/setup, then through the API: three
channels, an API key, a subscription plan, a redemption code, a demo Epay
configuration with a 20% discount on the 100 preset (so the wallet shows
the top-up presets; nothing is ever paid), and two scoped access tokens
(one expiring, one without expiry) after password verification. Through
SQLite: a second user, fixed timestamps on the tokens and usage logs on
2026-09-25 (UTC), so copies of the database render the same pages on
every build.
"""
import json
import sqlite3
import sys
import urllib.request

BASE = sys.argv[1]
DB = sys.argv[2]
PASSWORD = 'DemoPass-2026!'
T0 = 1790337600  # 2026-09-25 12:00:00 UTC


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


def option(key, value):
    call('PUT', '/api/option/', {'key': key, 'value': value}, auth)


def proof(scope, context=None):
    body = {'method': 'password', 'scope': scope, 'password': PASSWORD}
    if context is not None:
        body['context'] = context
    result = call('POST', '/api/verify', body, auth)
    return (result.get('data') or {}).get('proof_token', '')


option('PayAddress', 'https://pay.example.com/')
option('EpayId', 'demo')
option('EpayKey', 'demo-key-not-real')
option('payment_setting.amount_discount', json.dumps({'100': 0.8}))
call('POST', '/api/option/payment_compliance', {'confirmed': True}, auth)

for name, models in (('demo-a', 'gpt-4o,gpt-4o-mini'),
                     ('demo-b', 'claude-sonnet-4-5'),
                     ('demo-c', 'gpt-4o')):
    call('POST', '/api/channel/', {'mode': 'single', 'channel': {
        'name': name, 'type': 1, 'key': 'sk-demo-not-real', 'base_url': '',
        'models': models, 'group': 'default', 'status': 1}}, auth)

call('POST', '/api/token/', {'name': 'demo-key', 'remain_quota': 500000,
                             'expired_time': -1, 'unlimited_quota': True}, auth)
call('POST', '/api/subscription/admin/plans', {'plan': {
    'title': 'Basic', 'subtitle': 'Light usage', 'price_amount': 9.9,
    'duration_unit': 'month', 'duration_value': 1, 'enabled': True,
    'sort_order': 10, 'total_amount': 5000000}}, auth)
call('POST', '/api/redemption/', {'name': 'demo-code', 'count': 1,
                                  'quota': 500000, 'expired_time': 0}, auth)

# Scoped access tokens (each creation is also an audit record), with scopes
# taken from the server's permission catalog.
catalog = call('GET', '/api/user/access_tokens/catalog', headers=auth).get('data') or {}
available = [f"{r['resource']}:{a['action']}"
             for g in catalog.get('groups', []) for r in g['resources']
             for a in r['actions']]
print('scopes:', available[:8])
for name, scopes, expires_at in (
    ('deploy script', available[:4], T0 + 30 * 86400),
    ('nightly report', available[4:6], 0),
):
    context = {'scopes': scopes, 'expires_at': expires_at}
    call('POST', '/api/user/access_tokens',
         {'name': name, 'scopes': scopes, 'expires_at': expires_at},
         {**auth, 'X-Security-Proof': proof('access_token.generate', context)})

con = sqlite3.connect(DB)
con.execute('UPDATE users SET quota = 110000000, used_quota = 1650000, '
            'request_count = 3 WHERE id = 1')
con.execute(
    'INSERT INTO users (username, password, display_name, role, status, '
    'email, quota, used_quota, request_count, "group", aff_code, '
    'created_at) VALUES ("alice", "x", "Alice", 1, 1, '
    '"alice@example.com", 2500000, 120000, 12, "default", "ALIC", ?)',
    (T0,))
con.execute('UPDATE tokens SET created_time = ?, accessed_time = ?', (T0, T0))
con.execute('UPDATE channels SET created_time = ?, test_time = ?, '
            'response_time = 850', (T0, T0 + 1800))
con.execute('UPDATE user_access_tokens SET created_at = ?', (T0 - 86400,))
con.execute('UPDATE user_access_tokens SET last_used_at = ?, '
            'last_used_ip = "203.0.113.7" WHERE name = "deploy script"',
            (T0 + 3600,))
rows = [
    (T0 + 600, 2, 'gpt-4o', 4210, 1250, 380, 3, 1, 3, ''),
    (T0 + 1200, 2, 'claude-sonnet-4-5', 9180, 2400, 910, 5, 1, 2, ''),
    (T0 + 1800, 5, 'gpt-4o-mini', 0, 0, 0, 1, 0, 1, 'upstream error: status 429'),
]
for created, kind, model, quota, prompt, completion, use_time, stream, channel, content in rows:
    other = json.dumps({'model_ratio': 1.25, 'completion_ratio': 4,
                        'group_ratio': 1, 'frt': 850,
                        'admin_info': {'use_channel': [str(channel)]}})
    con.execute(
        'INSERT INTO logs (user_id, created_at, type, content, username, '
        'token_name, model_name, quota, prompt_tokens, completion_tokens, '
        'use_time, is_stream, channel_id, token_id, "group", ip, other) '
        'VALUES (1, ?, ?, ?, "admin", "demo-key", ?, ?, ?, ?, ?, ?, ?, 1, '
        '"default", "203.0.113.7", ?)',
        (created, kind, content, model, quota, prompt, completion, use_time,
         stream, channel, other))
con.commit()
con.close()
print('SEED_DONE')
