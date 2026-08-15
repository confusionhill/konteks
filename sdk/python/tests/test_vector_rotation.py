import unittest
import sys
import os
import numpy as np

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))

from contextkit_sdk.vector_rotation import VectorRotator

class TestVectorRotation(unittest.TestCase):

    def test_orthonormal_matrix_properties(self):
        dim = 128
        rotator = VectorRotator(dimension=dim)
        
        R = rotator.matrix
        # Check R^T * R = I
        identity_approx = np.dot(R.T, R)
        expected_identity = np.eye(dim)

        np.testing.assert_allclose(identity_approx, expected_identity, atol=1e-6)

    def test_cosine_similarity_preservation(self):
        dim = 64
        rotator = VectorRotator(dimension=dim)

        # Generate two random raw vectors u and v
        u = np.random.randn(dim)
        v = np.random.randn(dim)

        # Raw cosine similarity
        raw_sim = np.dot(u, v) / (np.linalg.norm(u) * np.linalg.norm(v))

        # Transformed vectors u_enc = R * u, v_enc = R * v
        u_enc = rotator.transform(u)
        v_enc = rotator.transform(v)

        enc_sim = np.dot(u_enc, v_enc) / (np.linalg.norm(u_enc) * np.linalg.norm(v_enc))

        # Expect exact cosine similarity match
        np.testing.assert_allclose(enc_sim, raw_sim, atol=1e-5)

if __name__ == "__main__":
    unittest.main()
