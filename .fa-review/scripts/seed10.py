"""Session 10 additions to the demo data (after seed.py, seed6.py, seed7.py, seed8.py).

Usage: python3 seed10.py <sqlite-db-path>

Adds hourly usage for the last seven days (two users, three models, two
demo nodes), so the dashboard, the traffic flow chart and the rankings
page have recent data. Demo values only; no request is ever sent.
"""
import sqlite3
import sys
import time

con = sqlite3.connect(sys.argv[1])
now = int(time.time()) // 3600 * 3600
models = [('gpt-4o', 1.0), ('claude-sonnet-4-5', 1.6), ('gemini-2.5-pro', 0.7)]
for hours_ago in range(0, 7 * 24, 3):
    ts = now - hours_ago * 3600
    for index, (model, factor) in enumerate(models):
        count = 3 + (hours_ago + index * 7) % 9
        for user_id, username, node in ((1, 'admin', 'node-a'), (2, 'demo-user', 'node-b')):
            con.execute(
                'INSERT INTO quota_data (user_id, username, model_name, created_at, use_group, '
                'token_id, channel_id, node_name, token_used, count, quota) '
                'VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)',
                (user_id, username, model, ts, 'default', user_id, 1 + index % 2, node,
                 count * 1500, count, int(count * 9000 * factor) // user_id))
con.commit()
print('SEED10_DONE')
