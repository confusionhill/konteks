import os
import json
import numpy as np
from typing import List, Union

class VectorRotator:
    """
    Distance-Preserving Secret Transformation Engine.
    Generates and applies an Orthonormal Rotation Matrix (R * v) where R^T * R = I,
    guaranteeing 100% exact mathematical cosine similarity preservation in pgvector.
    """

    def __init__(self, dimension: int = 1536, matrix: Union[np.ndarray, None] = None):
        self.dimension = dimension
        if matrix is not None:
            if matrix.shape != (dimension, dimension):
                raise ValueError(f"Invalid matrix shape {matrix.shape}; expected ({dimension}, {dimension})")
            self.matrix = matrix
        else:
            self.matrix = self._generate_orthonormal_matrix(dimension)

    @staticmethod
    def _generate_orthonormal_matrix(dim: int) -> np.ndarray:
        """
        Generates an orthonormal matrix R in R^(dim x dim) via QR decomposition.
        """
        H = np.random.randn(dim, dim)
        Q, R = np.linalg.qr(H)
        d = np.diag(R)
        ph = d / np.abs(d)
        Q = Q * ph
        return Q

    def transform(self, vector: Union[List[float], np.ndarray]) -> List[float]:
        """
        Transforms raw vector v -> v_enc = R * v
        """
        v_arr = np.array(vector, dtype=np.float64)
        if len(v_arr) != self.dimension:
            raise ValueError(f"Vector dimension {len(v_arr)} does not match rotator dimension {self.dimension}")

        rotated = np.dot(self.matrix, v_arr)
        return rotated.tolist()

    def save_matrix(self, filepath: str) -> None:
        """
        Saves the orthonormal rotation matrix to a file.
        """
        data = {
            "dimension": self.dimension,
            "matrix": self.matrix.tolist()
        }
        with open(filepath, 'w') as f:
            json.dump(data, f)

    @classmethod
    def load_matrix(cls, filepath: str) -> "VectorRotator":
        """
        Loads an orthonormal rotation matrix from a file.
        """
        with open(filepath, 'r') as f:
            data = json.load(f)
        matrix = np.array(data["matrix"], dtype=np.float64)
        return cls(dimension=data["dimension"], matrix=matrix)
