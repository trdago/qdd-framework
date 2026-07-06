import os
import yaml

cert_dir = ".qdd/core/certification"
for f in os.listdir(cert_dir):
    if not f.endswith(".yaml"):
        continue
    path = os.path.join(cert_dir, f)
    with open(path, "r") as file:
        data = yaml.safe_load(file)
    
    if "active" not in data:
        data["active"] = True
        
    if "tags" not in data:
        if "ISO" in f or "WCAG" in f:
            data["tags"] = ["frontend", "vue", "ui"]
        elif "OWASP" in f:
            data["tags"] = ["frontend", "backend", "core"]
        else:
            data["tags"] = ["core"]
            
    with open(path, "w") as file:
        yaml.dump(data, file, sort_keys=False)
