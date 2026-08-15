from setuptools import setup, find_packages

setup(
    name="contextkit-sdk",
    version="0.1.0",
    description="Official Python SDK & Zero-Trust E2EE Sidecar Engine for ContextKit",
    author="ContextKit",
    packages=find_packages(),
    install_requires=[
        "requests>=2.28.0",
        "cryptography>=38.0.0",
        "numpy>=1.22.0",
    ],
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
    ],
    python_requires=">=3.8",
)
