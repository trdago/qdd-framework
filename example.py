def procesar_edad(edad):
    if edad < 0:
        return "Edad inválida"
    
    if edad < 18:
        return "Menor de edad"
    
    return "Mayor de edad"
