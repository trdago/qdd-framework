import unittest
import os
import shutil
from engine import render_docs
from main import create_agnostic_template

class TestGoldenSets(unittest.TestCase):
    def setUp(self):
        self.output_dir = "/tmp/qdd_goldenset_docs"
        os.makedirs(self.output_dir, exist_ok=True)
        self.template_path = os.path.join(self.output_dir, "template.docx")
        
        # Generar la plantilla base para usarla en los tests
        create_agnostic_template(self.template_path)
        self.goldenset_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), "goldensets", "docs_engine")

    def tearDown(self):
        shutil.rmtree(self.output_dir, ignore_errors=True)

    def test_happy_path(self):
        """Valida que un JSON estándar se renderice correctamente."""
        json_path = os.path.join(self.goldenset_dir, "happy_path.json")
        out_path = os.path.join(self.output_dir, "out_happy.docx")
        
        result = render_docs(self.template_path, json_path, out_path)
        self.assertEqual(result["status"], "success")
        self.assertTrue(os.path.exists(out_path))

    def test_bad_path(self):
        """Valida que la omisión de listas (endpoints, db) no rompa el motor."""
        json_path = os.path.join(self.goldenset_dir, "bad_path.json")
        out_path = os.path.join(self.output_dir, "out_bad.docx")
        
        result = render_docs(self.template_path, json_path, out_path)
        self.assertEqual(result["status"], "success")  # Engine should autofill missing lists
        self.assertTrue(os.path.exists(out_path))

    def test_edge_case_giant(self):
        """Valida que el motor escale y no desborde con 500 endpoints y 500 tablas."""
        json_path = os.path.join(self.goldenset_dir, "edge_case_giant.json")
        out_path = os.path.join(self.output_dir, "out_giant.docx")
        
        result = render_docs(self.template_path, json_path, out_path)
        self.assertEqual(result["status"], "success")
        self.assertTrue(os.path.exists(out_path))
        
        # Validar que el archivo se generó y pesa más de 0 bytes
        self.assertGreater(os.path.getsize(out_path), 1000)

if __name__ == "__main__":
    unittest.main(verbosity=2)
