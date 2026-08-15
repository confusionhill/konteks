import base64
import json
import os
from typing import Dict, Any, Union
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
from cryptography.hazmat.primitives.kdf.hkdf import HKDF
from cryptography.hazmat.primitives import hashes

class ContextCrypto:
    """
    Zero-Knowledge Client Encryption Engine implementing HKDF room key derivation
    and AES-256-GCM envelope encryption.
    """

    @staticmethod
    def derive_room_key(master_key: str, room_id: str) -> bytes:
        """
        Derives a unique 256-bit room key (K_room) using HKDF SHA-256:
        K_room = HKDF-Expand(K_master, "konteks-room-" + room_id)
        """
        master_bytes = master_key.encode('utf-8')
        if len(master_bytes) < 32:
            master_bytes = master_bytes.ljust(32, b'\0')
        else:
            master_bytes = master_bytes[:32]

        info = f"konteks-room-{room_id}".encode('utf-8')
        hkdf = HKDF(
            algorithm=hashes.SHA256(),
            length=32,
            salt=None,
            info=info,
        )
        return hkdf.derive(master_bytes)

    @staticmethod
    def encrypt_payload(room_key: bytes, plaintext: str) -> str:
        """
        Encrypts plaintext string with AES-256-GCM and returns a Base64-encoded container string.
        """
        aesgcm = AESGCM(room_key)
        nonce = os.urandom(12)
        ciphertext = aesgcm.encrypt(nonce, plaintext.encode('utf-8'), None)

        container = {
            "version": "v1",
            "alg": "AES-256-GCM",
            "nonce": base64.b64encode(nonce).decode('utf-8'),
            "ciphertext": base64.b64encode(ciphertext).decode('utf-8')
        }

        encoded_json = json.dumps(container)
        return f"enc_{base64.b64encode(encoded_json.encode('utf-8')).decode('utf-8')}"

    @staticmethod
    def decrypt_payload(room_key: bytes, ciphertext_str: str) -> str:
        """
        Decrypts an AES-256-GCM Base64-encoded container string.
        """
        if not ciphertext_str:
            return ""

        raw_str = ciphertext_str.strip()
        if raw_str.startswith("enc_"):
            raw_str = raw_str[4:]

        try:
            json_bytes = base64.b64decode(raw_str)
            container = json.loads(json_bytes.decode('utf-8'))
            
            nonce = base64.b64decode(container["nonce"])
            ciphertext = base64.b64decode(container["ciphertext"])

            aesgcm = AESGCM(room_key)
            decrypted_bytes = aesgcm.decrypt(nonce, ciphertext, None)
            return decrypted_bytes.decode('utf-8')
        except Exception as e:
            # Fallback for plain base64 or unencrypted strings
            try:
                decoded = base64.b64decode(raw_str).decode('utf-8')
                return decoded
            except Exception:
                return ciphertext_str
