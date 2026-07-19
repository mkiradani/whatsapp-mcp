"""Guardián: la resolución LID<->teléfono debe leer de whatsmeow_lid_map.

Bug (2026-07-16): _lid_to_phone/_phone_to_lids resolvían SOLO desde aliases.json,
que tenía 2 entradas escritas a mano, mientras que el bridge ya mantiene
whatsmeow_lid_map (whatsapp.db) con cientos. Resultado: 443 de 445 contactos no
se resolvían — p. ej. un @lid entrante no se unía con su chat @s.whatsapp.net.

RED-antes-de-GREEN: contra el código con el bug, resolver un LID que sólo está en
whatsmeow_lid_map devuelve None.

Datos de ejemplo ficticios. Correr: uv run python -m unittest test_lid_resolution -v
"""
import os
import sqlite3
import tempfile
import unittest

import whatsapp


class LidResolutionTest(unittest.TestCase):
    def setUp(self):
        self._dir = tempfile.TemporaryDirectory()
        d = self._dir.name
        self.wa_db = os.path.join(d, "whatsapp.db")
        self.aliases = os.path.join(d, "aliases.json")

        conn = sqlite3.connect(self.wa_db)
        conn.execute("CREATE TABLE whatsmeow_lid_map (lid TEXT PRIMARY KEY, pn TEXT UNIQUE NOT NULL)")
        # Un contacto que SOLO vive en el lid_map del bridge, nunca en aliases.json.
        conn.execute("INSERT INTO whatsmeow_lid_map (lid, pn) VALUES ('99988877766', '34611122233')")
        conn.commit()
        conn.close()

        # Redirige el módulo a las rutas temporales y limpia cachés.
        self._orig_wa = getattr(whatsapp, "WHATSAPP_DB_PATH", None)
        self._orig_al = whatsapp.ALIASES_PATH
        whatsapp.WHATSAPP_DB_PATH = self.wa_db
        whatsapp.ALIASES_PATH = self.aliases
        self._reset_caches()

    def tearDown(self):
        if self._orig_wa is not None:
            whatsapp.WHATSAPP_DB_PATH = self._orig_wa
        whatsapp.ALIASES_PATH = self._orig_al
        self._reset_caches()
        self._dir.cleanup()

    def _reset_caches(self):
        whatsapp._aliases_cache = None
        if hasattr(whatsapp, "_lid_map_cache"):
            whatsapp._lid_map_cache = None

    def _write_aliases(self, mapping):
        import json
        with open(self.aliases, "w") as f:
            json.dump({"lid_to_phone": mapping}, f)
        self._reset_caches()

    def test_resuelve_lid_desde_lid_map(self):
        # El corazón del bug: sin aliases.json, el LID vive sólo en lid_map.
        self.assertEqual(whatsapp._lid_to_phone("99988877766"), "34611122233")

    def test_phone_to_lids_desde_lid_map(self):
        self.assertEqual(whatsapp._phone_to_lids("34611122233"), ["99988877766"])

    def test_resuelve_todos_los_jids(self):
        jids = whatsapp._resolve_all_chat_jids("99988877766@lid")
        self.assertIn("34611122233@s.whatsapp.net", jids)

    def test_aliases_sigue_siendo_override_manual(self):
        # aliases.json debe poder forzar un mapeo que lid_map no tiene.
        self._write_aliases({"55544433322": "34699988877"})
        self.assertEqual(whatsapp._lid_to_phone("55544433322"), "34699988877")

    def test_sin_whatsapp_db_no_revienta(self):
        whatsapp.WHATSAPP_DB_PATH = os.path.join(self._dir.name, "no-existe.db")
        self._reset_caches()
        # Degrada limpio: un LID desconocido no resuelve, pero no lanza.
        self.assertIsNone(whatsapp._lid_to_phone("00000000000"))


if __name__ == "__main__":
    unittest.main()
