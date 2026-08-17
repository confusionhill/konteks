"""Persistent local state for the ContextKit Claude Code adapter.

Two kinds of state, deliberately separated:

* Non-secret configuration is stored in ``${CLAUDE_PLUGIN_DATA}/state.json``
  (falling back to ``~/.contextkit/state.json`` when the env var is absent,
  e.g. when the adapter is used outside Claude Code). This directory is
  created by Claude Code on first use, survives plugin updates, and is removed
  on uninstall.
* The secret (the project API key) is stored with the macOS Keychain via
  ``security(1)`` on macOS, or in a 0600 file on platforms without a keychain.

Every write is atomic (write to a temp file, fsync, ``os.replace``).

Room bindings are keyed by the *project directory* (``CLAUDE_PROJECT_DIR``, or
the working directory as a fallback). This is what lets two different Claude
Code installations on the same machine use different rooms with the same API
key.
"""

from __future__ import annotations

import hashlib
import json
import os
import shutil
import subprocess
import sys
import tempfile
import uuid

STATE_VERSION = 1
KEY_SERVICE = "contextkit"
KEY_ACCOUNT = "contextkit-claude-plugin"
DEFAULT_REMOTE_URL = "https://mcp.contextkit.app/mcp"

# Maximum size of the api_keys.key value we ever read back from the keychain
# before refusing to log or cache it.
_LOGGABLE_KEY_PREFIX = 4


def state_dir() -> str:
    """Return the directory that holds non-secret plugin state."""
    env_dir = os.environ.get("CLAUDE_PLUGIN_DATA") or os.environ.get("CONTEXTKIT_STATE_DIR")
    if env_dir:
        return env_dir
    return os.path.join(os.path.expanduser("~"), ".contextkit")


def state_path(dirpath: str | None = None) -> str:
    return os.path.join(dirpath or state_dir(), "state.json")


def remote_url(cfg: dict | None = None) -> str:
    """Resolve the remote MCP endpoint: env var beats state file beats default."""
    env_url = os.environ.get("CONTEXTKIT_REMOTE_URL")
    if env_url:
        return env_url
    if cfg and cfg.get("remote_url"):
        return cfg["remote_url"]
    return DEFAULT_REMOTE_URL


def _ensure_dir(path: str) -> None:
    os.makedirs(path, exist_ok=True)


def default_config() -> dict:
    return {
        "version": STATE_VERSION,
        "installation_id": "",
        "remote_url": "",
        "project_id": "",
        "project_name": "",
        "bindings": [],
    }


def load_config(dirpath: str | None = None) -> dict:
    """Load state.json, tolerating missing or corrupt files."""
    cfg = default_config()
    path = state_path(dirpath)
    if not os.path.exists(path):
        return cfg
    try:
        with open(path, "r", encoding="utf-8") as fh:
            data = json.load(fh)
        if isinstance(data, dict):
            cfg.update({key: value for key, value in data.items() if key in cfg})
    except (OSError, ValueError):
        pass
    return cfg


def save_config(cfg: dict, dirpath: str | None = None) -> None:
    """Atomically persist state.json (tmp file + fsync + os.replace)."""
    _ensure_dir(dirpath or state_dir())
    path = state_path(dirpath)
    tmp_fd, tmp_path = tempfile.mkstemp(dir=os.path.dirname(path), suffix=".tmp")
    try:
        with os.fdopen(tmp_fd, "w", encoding="utf-8") as fh:
            json.dump(cfg, fh, indent=2, ensure_ascii=False)
            fh.flush()
            os.fsync(fh.fileno())
        os.replace(tmp_path, path)
    except BaseException:
        try:
            os.unlink(tmp_path)
        except OSError:
            pass
        raise


def clear_config(dirpath: str | None = None) -> None:
    try:
        os.unlink(state_path(dirpath))
    except OSError:
        pass


def ensure_installation_id(cfg: dict) -> str:
    if not cfg.get("installation_id"):
        cfg["installation_id"] = "inst_" + uuid.uuid4().hex[:20]
        save_config(cfg)
    return cfg["installation_id"]


# --------------------------------------------------------------------------- #
# Project-keyed room bindings
# --------------------------------------------------------------------------- #


def project_dir() -> str:
    return os.environ.get("CLAUDE_PROJECT_DIR") or os.getcwd()


def project_hash(path: str | None = None) -> str:
    path = path or project_dir()
    return hashlib.sha256(os.path.abspath(path).encode("utf-8")).hexdigest()


def get_binding(cfg: dict, path: str | None = None) -> dict | None:
    key = project_hash(path)
    for binding in cfg.get("bindings", []):
        if binding.get("project_dir_hash") == key:
            return binding
    return None


def set_binding(cfg: dict, room_id: str, room_name: str = "", path: str | None = None) -> dict:
    key = project_hash(path)
    cfg["bindings"] = [
        binding
        for binding in cfg.get("bindings", [])
        if binding.get("project_dir_hash") != key
    ]
    binding = {"project_dir_hash": key, "room_id": room_id, "room_name": room_name or ""}
    cfg["bindings"].append(binding)
    save_config(cfg)
    return binding


def unset_binding(cfg: dict, path: str | None = None) -> None:
    key = project_hash(path)
    cfg["bindings"] = [
        binding
        for binding in cfg.get("bindings", [])
        if binding.get("project_dir_hash") != key
    ]
    save_config(cfg)


# --------------------------------------------------------------------------- #
# Secret storage
# --------------------------------------------------------------------------- #


class KeyStore:
    """Abstract secret store. Subclasses implement get/set/delete."""

    def get(self) -> str | None:
        raise NotImplementedError

    def set(self, value: str) -> None:
        raise NotImplementedError

    def delete(self) -> None:
        raise NotImplementedError

    def has(self) -> bool:
        try:
            return self.get() is not None
        except OSError:
            return False


class FileKeyStore(KeyStore):
    """0600 API key file under the state directory (non-macOS fallback)."""

    def __init__(self, dirpath: str):
        self._path = os.path.join(dirpath, "api_key")

    def get(self) -> str | None:
        try:
            with open(self._path, "r", encoding="utf-8") as fh:
                value = fh.read().strip()
            return value or None
        except OSError:
            return None

    def set(self, value: str) -> None:
        _ensure_dir(os.path.dirname(self._path))
        tmp_fd, tmp_path = tempfile.mkstemp(dir=os.path.dirname(self._path), suffix=".tmp")
        try:
            with os.fdopen(tmp_fd, "w", encoding="utf-8") as fh:
                fh.write(value)
                fh.flush()
                os.fsync(fh.fileno())
            os.chmod(tmp_path, 0o600)
            os.replace(tmp_path, self._path)
            os.chmod(self._path, 0o600)
        except BaseException:
            try:
                os.unlink(tmp_path)
            except OSError:
                pass
            raise

    def delete(self) -> None:
        try:
            os.unlink(self._path)
        except OSError:
            pass


class KeychainKeyStore(KeyStore):
    """macOS Keychain via security(1). Uses -U to update existing entries."""

    def __init__(self, service: str = KEY_SERVICE, account: str = KEY_ACCOUNT):
        self.service = service
        self.account = account

    @staticmethod
    def _available() -> bool:
        return sys.platform == "darwin" and shutil.which("security") is not None

    def get(self) -> str | None:
        if not self._available():
            return None
        try:
            result = subprocess.run(
                ["security", "find-generic-password",
                 "-a", self.account, "-s", self.service, "-w"],
                capture_output=True, text=True, check=False,
            )
        except OSError:
            return None
        if result.returncode != 0:
            return None
        return result.stdout.strip() or None

    def set(self, value: str) -> None:
        if not self._available():
            raise OSError("security(1) is not available on this platform")
        result = subprocess.run(
            ["security", "add-generic-password", "-U",
             "-a", self.account, "-s", self.service, "-w", value],
            capture_output=True, text=True, check=False,
        )
        if result.returncode != 0:
            raise OSError(result.stderr.strip() or "keychain add failed")

    def delete(self) -> None:
        if not self._available():
            return
        try:
            subprocess.run(
                ["security", "delete-generic-password",
                 "-a", self.account, "-s", self.service],
                capture_output=True, text=True, check=False,
            )
        except OSError:
            pass


def default_key_store(dirpath: str | None = None) -> KeyStore:
    """Pick the platform key store, unless the user forces the file store.

    ``CONTEXTKIT_KEY_STORE=file`` forces the 0600-file backend on any platform
    (useful for tests and for machines without a supported keychain).
    """
    if os.environ.get("CONTEXTKIT_KEY_STORE") == "file":
        return FileKeyStore(dirpath or state_dir())
    if sys.platform == "darwin":
        return KeychainKeyStore()
    return FileKeyStore(dirpath or state_dir())