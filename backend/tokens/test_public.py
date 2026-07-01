#!/usr/bin/env python3
"""Test public API token - returns a mock token with the parameter embedded."""
import sys
import os


def main():
    if len(sys.argv) < 2:
        print("Usage: test_public.py <param>", file=sys.stderr)
        sys.exit(1)
    param = sys.argv[1]
    token = os.environ.get(
        "TEST_PUBLIC_TOKEN",
        f"pub_{param}_a1b2c3d4e5f6g7h8i9j0"
    )
    print(token)


if __name__ == "__main__":
    main()
