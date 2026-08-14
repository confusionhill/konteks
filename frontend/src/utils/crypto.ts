// Web Crypto API Helper for Client-Side In-Browser Decryption

export async function decryptAESGCM(
  ciphertextB64: string,
  nonceB64: string,
  secretKeyStr: string
): Promise<string> {
  const encoder = new TextEncoder();
  const decoder = new TextDecoder();

  // Handle fallback mock ciphertexts if any
  if (ciphertextB64.startsWith('enc_')) {
    try {
      return atob(ciphertextB64.slice(4));
    } catch {
      return ciphertextB64;
    }
  }

  // Decode Base64 Nonce & Ciphertext
  const nonceBuf = Uint8Array.from(atob(nonceB64), (c) => c.charCodeAt(0));
  const cipherBuf = Uint8Array.from(atob(ciphertextB64), (c) => c.charCodeAt(0));

  // Prepare Key Candidates:
  // Candidate 1: Raw 32-byte key (Python SDK standard: sliced/padded to 32 bytes)
  const fullBytes = encoder.encode(secretKeyStr);
  const raw32 = new Uint8Array(32);
  raw32.set(fullBytes.subarray(0, Math.min(fullBytes.length, 32)));

  // Candidate 2: SHA-256 digested key
  const sha256Key = await window.crypto.subtle.digest('SHA-256', fullBytes);

  const keyCandidates = [raw32, new Uint8Array(sha256Key)];

  for (const rawKeyBuf of keyCandidates) {
    try {
      const cryptoKey = await window.crypto.subtle.importKey(
        'raw',
        rawKeyBuf,
        { name: 'AES-GCM' },
        false,
        ['decrypt']
      );

      const decryptedBuf = await window.crypto.subtle.decrypt(
        { name: 'AES-GCM', iv: nonceBuf },
        cryptoKey,
        cipherBuf
      );

      return decoder.decode(decryptedBuf);
    } catch {
      // Try next key format candidate
    }
  }

  throw new Error('Failed to decrypt payload with provided secret key');
}
