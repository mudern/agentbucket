#!/usr/bin/env python3
"""GitHub token resolver.

Reads GITHUB_TOKEN from the environment. Falls back to a test token with
the parameter (typically a repository name) embedded in the note portion.
"""
import sys
import os


def main():
    if len(sys.argv) < 2:
        print("Usage: github_token.py <repo>", file=sys.stderr)
        sys.exit(1)
    param = sys.argv[1]
    token = os.environ.get(
        "GITHUB_TOKEN",
        f"github_pat_{param}_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0"
    )
    print(token)


if __name__ == "__main__":
    main()
