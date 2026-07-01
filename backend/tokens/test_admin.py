#!/usr/bin/env python3
"""Test admin API token - returns a mock admin token with the parameter embedded."""
import sys
import os


def main():
    if len(sys.argv) < 2:
        print("Usage: test_admin.py <param>", file=sys.stderr)
        sys.exit(1)
    param = sys.argv[1]
    token = os.environ.get(
        "TEST_ADMIN_TOKEN",
        f"adm_{param}_9z8y7x6w5v4u3t2s1r0qp"
    )
    print(token)


if __name__ == "__main__":
    main()
