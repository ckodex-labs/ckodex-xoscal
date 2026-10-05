#!/usr/bin/env python3
"""Block publication before exceeding the provider's published-site limit."""
import json
from pathlib import Path
import sys
from release_inventory import files

MAX_SITE_BYTES = 1_000_000_000


def check(root, limit=MAX_SITE_BYTES):
    names = files(root)
    total = sum((root / name).stat().st_size for name in names)
    # Leave room for the final manifest and detached proof envelope; final
    # verification checks the complete tree again before public promotion.
    if total > limit:
        raise ValueError(f'Pages capacity exceeded: {total} > {limit} bytes')
    return {'scope': 'pages-capacity', 'bytes': total, 'files': len(names), 'limit': limit}


if __name__ == '__main__':
    print(json.dumps(check(Path(sys.argv[1])), sort_keys=True))
