import unittest
from math_utils import suma

class TestSuma(unittest.TestCase):
    def test_suma_correcta(self):
        self.assertEqual(suma(2, 3), 5)
        self.assertEqual(suma(-1, 1), 0)
        self.assertEqual(suma(2.5, 2.5), 5.0)

    def test_suma_error_tipo(self):
        with self.assertRaises(TypeError):
            suma("a", 2)
        with self.assertRaises(TypeError):
            suma(2, "b")

if __name__ == '__main__':
    unittest.main()
