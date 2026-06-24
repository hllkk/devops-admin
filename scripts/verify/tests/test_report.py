import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import probe
import report


def _session_with(*errors):
    s = probe.ProbeSession()
    for e in errors:
        s.add(e)
    return s


class TestRender(unittest.TestCase):
    def test_all_pass_report(self):
        s = probe.ProbeSession()  # 无错误
        checks = [report.CheckResult("列表渲染", True), report.CheckResult("按钮可见", True)]
        out = report.render("网盘页", smoke_passed=True, smoke_detail="登录✓ 主页✓",
                            checks=checks, probe_obj=s, screenshot_path="/tmp/a.png")
        self.assertIn("验证报告: 网盘页", out)
        self.assertIn("基线冒烟:    ✅ PASS", out)
        self.assertIn("功能检查:    ✅ PASS", out)
        self.assertNotIn("错误清单", out)

    def test_failed_report_lists_errors(self):
        s = _session_with(
            probe.VerifError(kind="console.error", message="boom", location="a.vue:1"),
            probe.VerifError(kind="business", message="POST /api/x → 业务码 5001", detail="业务码 5001"),
        )
        checks = [report.CheckResult("列表渲染", False, "期望≥1 实际0")]
        out = report.render("网盘页", smoke_passed=True, smoke_detail="ok",
                            checks=checks, probe_obj=s, screenshot_path="/tmp/a.png")
        self.assertIn("功能检查:    ❌ FAIL", out)
        self.assertIn("错误清单", out)
        self.assertIn("[console.error] boom", out)
        self.assertIn("[business] POST /api/x", out)
        self.assertIn("列表渲染: ❌ 期望≥1 实际0", out)

    def test_screenshot_path_unique_and_in_reports_dir(self):
        p = report.screenshot_path("disk-page")
        self.assertIn("reports/", p)
        self.assertTrue(p.endswith(".png"))


if __name__ == "__main__":
    unittest.main()
