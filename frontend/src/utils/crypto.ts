// Web Crypto API Helper for Client-Side In-Browser Decryption

export async function decryptAESGCM(
  ciphertextB64: string,
  nonceB64: string,
  secretKeyStr: string
): Promise<string> {
  if (!ciphertextB64) return '';

  const encoder = new TextEncoder();
  const decoder = new TextDecoder();

  // Strip enc_ prefix if present
  let rawCipherB64 = ciphertextB64.trim();
  if (rawCipherB64.startsWith('enc_')) {
    rawCipherB64 = rawCipherB64.slice(4);
  }

  // Handle Nonce: default to 12-byte zero if missing/empty
  let nonceBuf: Uint8Array;
  if (nonceB64 && nonceB64.trim() !== '') {
    try {
      const cleanNonce = nonceB64.startsWith('nonce_') ? 'nonce_static' : nonceB64;
      if (cleanNonce === 'nonce_static') {
        nonceBuf = encoder.encode('nonce_static').subarray(0, 12);
      } else {
        nonceBuf = Uint8Array.from(atob(cleanNonce), (c) => c.charCodeAt(0));
      }
    } catch {
      nonceBuf = new Uint8Array(12);
    }
  } else {
    nonceBuf = new Uint8Array(12);
  }

  // Decode Base64 Ciphertext
  let cipherBuf: Uint8Array;
  try {
    cipherBuf = Uint8Array.from(atob(rawCipherB64), (c) => c.charCodeAt(0));
  } catch {
    // If not valid base64, return original
    return ciphertextB64;
  }

  // Prepare Key Candidates:
  // Candidate 1: Raw 32-byte key (Python SDK standard: sliced/padded to 32 bytes)
  const fullBytes = encoder.encode(secretKeyStr);
  const raw32 = new Uint8Array(32);
  raw32.set(fullBytes.subarray(0, Math.min(fullBytes.length, 32)));

  // Candidate 2: Hex decoded key if 64 hex chars
  let hexBytes: Uint8Array | null = null;
  if (secretKeyStr.length === 64 && /^[0-9a-fA-F]+$/.test(secretKeyStr)) {
    const match = secretKeyStr.match(/.{1,2}/g);
    if (match) {
      hexBytes = new Uint8Array(match.map((byte) => parseInt(byte, 16)));
    }
  }

  // Candidate 3: SHA-256 digested key
  const sha256Key = await window.crypto.subtle.digest('SHA-256', fullBytes);

  const keyCandidates: ArrayBuffer[] = [
    raw32.buffer as ArrayBuffer,
    new Uint8Array(sha256Key).buffer as ArrayBuffer,
  ];
  if (hexBytes && hexBytes.length === 32) {
    keyCandidates.push(hexBytes.buffer as ArrayBuffer);
  }

  // 1. Try AES-GCM decryption with candidates
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
        { name: 'AES-GCM', iv: nonceBuf as any },
        cryptoKey,
        cipherBuf as any
      );

      const result = decoder.decode(decryptedBuf);
      if (result) return result;
    } catch {
      // Try next key candidate
    }
  }

  // 2. Fallback: Check if rawCipherB64 was plain Base64 string (non-AES mock)
  try {
    const decodedStr = atob(rawCipherB64);
    // If printable string, return it
    if (/^[\x20-\x7E\s\u00A0-\uFFFF]+$/.test(decodedStr)) {
      return decodedStr;
    }
  } catch {}

  throw new Error('Failed to decrypt payload with provided secret key');
}
