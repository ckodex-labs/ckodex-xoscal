#!/usr/bin/env python3
"""Verify exact frozen transport under the same hosted manifest identity."""
import json
from pathlib import Path
import subprocess
import sys
from attestation_policy import verification_flags
from release_inventory import digest, write

transport, bundle, tag, revision, receipt = sys.argv[1:]
result = subprocess.run(["gh", "attestation", "verify", transport] + verification_flags(bundle, tag, revision),
                        check=True, capture_output=True, text=True)
verified = json.loads(result.stdout)
if not isinstance(verified, list) or not verified:
    raise ValueError("verifier supplied no successful transport attestation")
write(Path(receipt), {"scope": "exact-transport-verification", "release_tag": tag,
                      "source_revision": revision, "transport_digest": digest(Path(transport)),
                      "bundle_digest": digest(Path(bundle)), "results": verified})
