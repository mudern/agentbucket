#!/usr/bin/env python3
"""Notion API token resolver.

Reads NOTION_TOKEN from the environment. Falls back to a test token with
the parameter (typically a workspace or integration name) embedded.
"""
import sys
import os


def main():
    if len(sys.argv) < 2:
        print("Usage: notion_token.py <integration>", file=sys.stderr)
        sys.exit(1)
    param = sys.argv[1]
    token = os.environ.get(
        "NOTION_TOKEN",
        f"ntn_{param}_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0"
    )
    print(token)


if __name__ == "__main__":
    main()
