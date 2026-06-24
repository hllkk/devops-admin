import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import config


class TestConfig(unittest.TestCase):
    def setUp(self):
        os.environ.pop("DEVOPS_TEST_USER", None)
        os.environ.pop("DEVOPS_TEST_PASS", None)

    def test_account_from_env_ok(self):
        os.environ["DEVOPS_TEST_USER"] = "User"
        os.environ["DEVOPS_TEST_PASS"] = "admin@123"
        acc = config.Account.from_env()
        self.assertEqual(acc.username, "User")
        self.assertEqual(acc.password, "admin@123")

    def test_account_missing_raises(self):
        with self.assertRaises(RuntimeError) as ctx:
            config.Account.from_env()
        self.assertIn("DEVOPS_TEST_USER", str(ctx.exception))

    def test_account_whitespace_only_raises(self):
        os.environ["DEVOPS_TEST_USER"] = "   "
        os.environ["DEVOPS_TEST_PASS"] = "x"
        with self.assertRaises(RuntimeError):
            config.Account.from_env()

    def test_is_success_code(self):
        self.assertTrue(config.is_success_code("0000"))
        self.assertFalse(config.is_success_code("7"))
        self.assertFalse(config.is_success_code("5001"))
        self.assertFalse(config.is_success_code(None))

    def test_check_services_reachable_failure(self):
        from unittest import mock
        with mock.patch("urllib.request.urlopen", side_effect=Exception("conn refused")):
            with self.assertRaises(RuntimeError) as ctx:
                config.check_services_reachable()
        self.assertIn("不可达", str(ctx.exception))


if __name__ == "__main__":
    unittest.main()
