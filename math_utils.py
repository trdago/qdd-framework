def suma(a, b):
    if not isinstance(a, (int, float)):
        raise TypeError("El parámetro 'a' debe ser un número")
    if not isinstance(b, (int, float)):
        raise TypeError("El parámetro 'b' debe ser un número")
    
    return a + b
