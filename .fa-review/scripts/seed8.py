"""Session 8 additions to the demo data (after seed.py, seed6.py, seed7.py).

Usage: python3 seed8.py <sqlite-db-path>

Gives the three demo channels a last response time, so the channels table
shows the response-time badge (demo values only; no request is ever sent).
"""
import sqlite3
import sys
import time

con = sqlite3.connect(sys.argv[1])
now = int(time.time())
for channel_id, ms in ((1, 1234), (2, 456), (3, 12345)):
    con.execute('UPDATE channels SET response_time = ?, test_time = ? WHERE id = ?',
                (ms, now, channel_id))
con.commit()
print('SEED8_DONE')
