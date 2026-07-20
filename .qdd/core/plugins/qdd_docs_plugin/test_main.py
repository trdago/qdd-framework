import unittest
import os
import json
from main import create_agnostic_template

class TestQDDDocsPlugin(unittest.TestCase):
    def setUp(self):
        self.test_output = "/tmp/Formato_Agnostico_Test.docx"
        
    def tearDown(self):
        if os.path.exists(self.test_output):
            os.remove(self.test_output)

    def test_create_template_success(self):
        """Golden Set: Valid generation of agnostic template"""
        result = create_agnostic_template(self.test_output)
        self.assertEqual(result["status"], "success")
        self.assertTrue(os.path.exists(self.test_output))
        
    def test_create_template_empty_path(self):
        """Error handling: Empty path provided"""
        result = create_agnostic_template("")
        self.assertEqual(result["status"], "error")
        self.assertEqual(result["message"], "output_path is empty")

if __name__ == '__main__':
    unittest.main()
