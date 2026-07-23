import sys
import json
import os
from docxtpl import DocxTemplate

def render_docs(template_path, json_path, output_path):
    """
    Renderiza el documento final aplicando reglas Zero-Else y guard clauses.
    """
    if not os.path.exists(template_path):
        return {"status": "error", "message": f"Template missing: {template_path}"}
        
    if not os.path.exists(json_path):
        return {"status": "error", "message": f"JSON missing: {json_path}"}

    try:
        doc = DocxTemplate(template_path)
        
        with open(json_path, 'r', encoding='utf-8') as f:
            context = json.load(f)
            
        # Zero-Else Guard Clauses for missing keys
        if "integrantes" not in context:
            context["integrantes"] = []
            
        if "endpoints" not in context:
            context["endpoints"] = []
            
        if "database" not in context:
            context["database"] = []
            
        doc.render(context)
        
        os.makedirs(os.path.dirname(output_path), exist_ok=True)
        doc.save(output_path)
        
        return {"status": "success", "message": output_path}
    except Exception as e:
        return {"status": "error", "message": str(e)}

def main():
    if len(sys.argv) < 4:
        print(json.dumps({"status": "error", "message": "Usage: engine.py <template> <json> <output>"}))
        sys.exit(1)
        
    result = render_docs(sys.argv[1], sys.argv[2], sys.argv[3])
    print(json.dumps(result))
    if result["status"] == "error":
        sys.exit(1)

if __name__ == "__main__":
    main()
