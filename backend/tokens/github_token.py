#!/usr/bin/env python3
"""GitHub token resolver."""
import sys
import hashlib
import time

repo = sys.argv[1] if len(sys.argv) > 1 else "default"
token = hashlib.sha256(f"github-{repo}-{int(time.time())}".encode()).hexdigest()[:40]
print(f"ghp_{token}")
