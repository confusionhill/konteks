import unittest
import sys
import os

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))

from contextkit_sdk.crypto import ContextCrypto

class TestCrypto(unittest.TestCase):

    def test_derive_room_key(self):
        master_key = "secret_master_key_12345678901234567890"
        room_1 = "room-uuid-1"
        room_2 = "room-uuid-2"

        key_1 = ContextCrypto.derive_room_key(master_key, room_1)
        key_2 = ContextCrypto.derive_room_key(master_key, room_2)

        self.assertEqual(len(key_1), 32)
        self.assertEqual(len(key_2), 32)
        self.assertNotEqual(key_1, key_2)

        # Determinism test
        key_1_again = ContextCrypto.derive_room_key(master_key, room_1)
        self.assertEqual(key_1, key_1_again)

    def test_encrypt_decrypt_roundtrip(self):
        master_key = "secret_master_key_12345678901234567890"
        room_id = "room-uuid-1"
        plaintext = "User's sensitive financial budget is $15,000 USD."

        room_key = ContextCrypto.derive_room_key(master_key, room_id)
        ciphertext_container = ContextCrypto.encrypt_payload(room_key, plaintext)

        self.assertTrue(ciphertext_container.startswith("enc_"))
        self.assertNotIn(plaintext, ciphertext_container)

        decrypted = ContextCrypto.decrypt_payload(room_key, ciphertext_container)
        self.assertEqual(decrypted, plaintext)

if __name__ == "__main__":
    unittest.main()
