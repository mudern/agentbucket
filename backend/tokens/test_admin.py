#!/usr/bin/env python3
"""Test admin API token - returns a mock token for testing."""
import sys
import hashlib
import time

scope = sys.argv[1] if len(sys.argv) > 1 else "default"
token = hashlib.sha256(f"admin-{scope}-{int(time.time())}".encode()).hexdigest()[:32]
print(f"adm-{token}")
