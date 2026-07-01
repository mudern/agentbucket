#!/usr/bin/env python3
"""Database read-only token resolver."""
import sys
import hashlib
import time

database = sys.argv[1] if len(sys.argv) > 1 else "default"
token = hashlib.sha256(f"db-{database}-{int(time.time())}".encode()).hexdigest()[:32]
print(f"db_ro_{token}")
