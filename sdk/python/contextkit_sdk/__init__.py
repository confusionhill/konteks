"""
ContextKit Python Client SDK & Zero-Trust E2EE Sidecar Engine
"""

from .client import ContextKitClient
from .crypto import ContextCrypto
from .vector_rotation import VectorRotator
from .health_worker import ClientHealthWorker

__all__ = [
    "ContextKitClient",
    "ContextCrypto",
    "VectorRotator",
    "ClientHealthWorker",
]
__version__ = "0.1.0"
