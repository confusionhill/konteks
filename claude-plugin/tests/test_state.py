import os
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from contextkit import state as state_mod  # noqa: E402


class StateFileTests(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self._tmp.cleanup)
        self.dir = self._tmp.name
        self.old_state_dir = os.environ.get("CONTEXTKIT_STATE_DIR")
        os.environ["CONTEXTKIT_STATE_DIR"] = self.dir

    def tearDown(self):
        if self.old_state_dir is None:
            os.environ.pop("CONTEXTKIT_STATE_DIR", None)
        else:
            os.environ["CONTEXTKIT_STATE_DIR"] = self.old_state_dir

    def test_load_defaults_when_missing(self):
        cfg = state_mod.load_config(self.dir)
        self.assertEqual(cfg["version"], 1)
        self.assertEqual(cfg["installation_id"], "")
        self.assertEqual(cfg["bindings"], [])

    def test_save_and_load_round_trip(self):
        cfg = state_mod.load_config(self.dir)
        cfg["project_id"] = "proj_x"
        state_mod.save_config(cfg, self.dir)
        reloaded = state_mod.load_config(self.dir)
        self.assertEqual(reloaded["project_id"], "proj_x")

    def test_corrupt_state_ignored(self):
        path = state_mod.state_path(self.dir)
        os.makedirs(self.dir, exist_ok=True)
        with open(path, "w", encoding="utf-8") as fh:
            fh.write("{not json")
        cfg = state_mod.load_config(self.dir)
        self.assertEqual(cfg["bindings"], [])

    def test_unknown_keys_not_merged(self):
        with open(state_mod.state_path(self.dir), "w", encoding="utf-8") as fh:
            fh.write('{"version": 1, "room_id": "room_9", "totally_unknown": true}')
        cfg = state_mod.load_config(self.dir)
        self.assertNotIn("totally_unknown", cfg)
        self.assertEqual(cfg["version"], 1)

    def test_project_hash_is_stable_and_distinct(self):
        self.assertEqual(state_mod.project_hash("/a/b"), state_mod.project_hash("/a/b"))
        self.assertNotEqual(state_mod.project_hash("/a/b"), state_mod.project_hash("/a/c"))

    def test_bindings_are_keyed_by_project(self):
        cfg = state_mod.load_config(self.dir)
        state_mod.set_binding(cfg, "room_a", "Alpha", path="/p/one")
        state_mod.set_binding(cfg, "room_b", "Beta", path="/p/two")
        self.assertEqual(state_mod.get_binding(cfg, "/p/one")["room_id"], "room_a")
        self.assertEqual(state_mod.get_binding(cfg, "/p/two")["room_id"], "room_b")
        # one binding per project: re-binding replaces
        state_mod.set_binding(cfg, "room_c", "Gamma", path="/p/one")
        self.assertEqual(len(cfg["bindings"]), 2)
        self.assertEqual(state_mod.get_binding(cfg, "/p/one")["room_id"], "room_c")

    def test_unset_binding(self):
        cfg = state_mod.load_config(self.dir)
        state_mod.set_binding(cfg, "room_a", "Alpha", path="/p/one")
        state_mod.unset_binding(cfg, "/p/one")
        self.assertIsNone(state_mod.get_binding(cfg, "/p/one"))

    def test_installation_id_created_once(self):
        os.environ["CLAUDE_PROJECT_DIR"] = self.dir
        try:
            cfg = state_mod.load_config(self.dir)
            first = state_mod.ensure_installation_id(cfg)
            second = state_mod.ensure_installation_id(cfg)
            self.assertEqual(first, second)
            self.assertTrue(first.startswith("inst_"))
        finally:
            os.environ.pop("CLAUDE_PROJECT_DIR", None)


class FileKeyStoreTests(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self._tmp.cleanup)
        self.store = state_mod.FileKeyStore(self._tmp.name)

    def test_set_get_delete(self):
        self.assertIsNone(self.store.get())
        self.store.set("ck_live_secret")
        self.assertEqual(self.store.get(), "ck_live_secret")
        self.assertTrue(self.store.has())
        self.store.delete()
        self.assertIsNone(self.store.get())

    def test_file_permissions_are_0600(self):
        import stat as stat_mod

        self.store.set("ck_live_secret")
        mode = stat_mod.S_IMODE(os.stat(os.path.join(self._tmp.name, "api_key")).st_mode)
        self.assertEqual(mode, 0o600)


if __name__ == "__main__":
    unittest.main()