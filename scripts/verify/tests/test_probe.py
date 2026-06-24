import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import probe


class TestProbeClassification(unittest.TestCase):
    def test_classify_console_error(self):
        self.assertEqual(probe.classify_console("error"), "console.error")

    def test_classify_console_warning(self):
        self.assertEqual(probe.classify_console("warning"), "console.warning")

    def test_classify_console_info_ignored(self):
        self.assertIsNone(probe.classify_console("info"))
        self.assertIsNone(probe.classify_console("log"))

    def test_parse_business_error_on_failure_code(self):
        self.assertEqual(
            probe.parse_business_error({"code": "5001", "msg": "权限不足"}),
            '业务码 5001: "权限不足"',
        )

    def test_parse_business_error_success_returns_none(self):
        self.assertIsNone(probe.parse_business_error({"code": "0000", "msg": "ok"}))

    def test_parse_business_error_non_dict_returns_none(self):
        self.assertIsNone(probe.parse_business_error("not an object"))
        self.assertIsNone(probe.parse_business_error(None))


class TestProbeSession(unittest.TestCase):
    def test_starts_empty(self):
        s = probe.ProbeSession()
        self.assertFalse(s.has_errors)
        self.assertEqual(len(s.errors), 0)

    def test_add_and_dedup(self):
        s = probe.ProbeSession()
        e1 = probe.VerifError(kind="console.error", message="boom")
        e2 = probe.VerifError(kind="console.error", message="boom")  # 重复
        e3 = probe.VerifError(kind="pageerror", message="boom")
        s.add(e1)
        s.add(e2)
        s.add(e3)
        self.assertTrue(s.has_errors)
        self.assertEqual(len(s.errors), 2)  # e1 与 e2 去重

    def test_summary_by_kind(self):
        s = probe.ProbeSession()
        s.add(probe.VerifError(kind="console.error", message="a"))
        s.add(probe.VerifError(kind="console.error", message="b"))
        s.add(probe.VerifError(kind="network", message="c"))
        summary = s.summary_by_kind()
        self.assertEqual(summary["console.error"], 2)
        self.assertEqual(summary["network"], 1)


if __name__ == "__main__":
    unittest.main()
