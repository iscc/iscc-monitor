"""Derive C2SP signed-note verifier keys for ISCC-Log hubs from their did:key pubkeys.

Maps each Hub's published multibase Ed25519 public key (z6Mk... in the Hub-List)
to the ``name+hash+base64`` verifier-key string that off-the-shelf tlog-tiles
tooling (Tessera fsck, x/mod/sumdb/note) needs. The signed-note ``name`` MUST
equal the checkpoint origin actually served by the Hub (e.g. ``sb0.iscc.id/log``),
not merely the bare domain.
"""

import base64
import hashlib
import os
import struct
import sys

_B58 = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
_ALG_ED25519 = 1


def b58decode(s):
    # type: (str) -> bytes
    """Decode a base58btc string to bytes (preserving leading-zero bytes)."""
    num = 0
    for ch in s:
        num = num * 58 + _B58.index(ch)
    body = num.to_bytes((num.bit_length() + 7) // 8, "big") if num else b""
    pad = len(s) - len(s.lstrip("1"))
    return b"\x00" * pad + body


def pubkey_from_did(z):
    # type: (str) -> bytes
    """Extract the 32-byte Ed25519 public key from a z6Mk multibase did:key."""
    raw = b58decode(z[1:])  # strip multibase 'z' prefix
    assert raw[0:2] == b"\xed\x01", f"not ed25519-pub multicodec: {raw[0:2].hex()}"
    return raw[2:34]


def key_id(name, pub):
    # type: (str, bytes) -> int
    """Return the signed-note 4-byte key id (SHA-256(name||0x0A||0x01||pub)[:4])."""
    h = hashlib.sha256()
    h.update(name.encode())
    h.update(b"\x0a")
    h.update(bytes([_ALG_ED25519]))
    h.update(pub)
    return struct.unpack(">I", h.digest()[:4])[0]


def verifier_key(name, pub):
    # type: (str, bytes) -> str
    """Return the signed-note verifier-key string ``name+hash+base64(0x01||pub)``."""
    encoded = bytes([_ALG_ED25519]) + pub
    return f"{name}+{key_id(name, pub):08x}+{base64.b64encode(encoded).decode()}"


# origin (= signed-note name actually served) -> published did:key pubkey
HUBS = {
    "sb0.iscc.id/log": "z6MkqbHELZopsq6eKrn6qxiAgRoVvwp2Vp7mPfKsvrbYmGwJ",
    "sb1.amlet.id/log": "z6MkiNW46AUjNmKTV2YNyFi9ANG9wbfYQQoUQgADGwScd9jk",
}

if __name__ == "__main__":
    outdir = os.path.join(os.path.dirname(os.path.abspath(__file__)), ".scratch")
    os.makedirs(outdir, exist_ok=True)
    for origin, did in HUBS.items():
        pub = pubkey_from_did(did)
        vk = verifier_key(origin, pub)
        path = os.path.join(outdir, origin.replace("/", "_") + ".pub")
        with open(path, "w", newline="") as f:
            f.write(vk)
        print(f"{origin}\t{vk}\t{path}")
    sys.stdout.flush()
