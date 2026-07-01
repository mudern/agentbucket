#!/usr/bin/env python3
"""Database connection-string resolver.

Reads INTERNAL_DB_URL from the environment. Falls back to a local SQLite
path with the parameter (typically a database name) used as the filename.
"""
import sys
import os


def main():
    if len(sys.argv) < 2:
        print("Usage: internal_db.py <dbname>", file=sys.stderr)
        sys.exit(1)
    param = sys.argv[1]
    token = os.environ.get(
        "INTERNAL_DB_URL",
        f"sqlite:///var/lib/agentbucket/{param}.db"
    )
    print(token)


if __name__ == "__main__":
    main()
