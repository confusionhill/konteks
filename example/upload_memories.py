import os
import json
import requests

CONFIG_PATH = "example/secrets/config.json"
BASE_URL = "http://localhost:8080"
PROJECT_ID = "4c15a993-785d-426e-8085-7ecf18564e8b"

# Read API Key & Auth Token
if not os.path.exists(CONFIG_PATH):
    print(f"Error: configuration file not found at {CONFIG_PATH}")
    exit(1)

with open(CONFIG_PATH, "r") as f:
    cfg = json.load(f)

api_key = cfg.get("konteks_api_key", "ck_live_dev_test_key_123")
headers = {
    "Authorization": f"Bearer {api_key}",
    "Content-Type": "application/json",
    "X-Project-ID": PROJECT_ID
}

memories_dir = "example/memories"
if not os.path.exists(memories_dir):
    print(f"Directory {memories_dir} not found.")
    exit(1)

print(f"🚀 Uploading sample Markdown Memory Artifacts to Konteks Project '{PROJECT_ID}'...")

for filename in sorted(os.listdir(memories_dir)):
    if filename.endswith(".md"):
        file_path = os.path.join(memories_dir, filename)
        rel_path = f"docs/{filename}"

        with open(file_path, "r", encoding="utf-8") as f:
            content = f.read()

        payload = {
            "file_path": rel_path,
            "title": filename.replace(".md", "").replace("_", " ").title(),
            "content": content
        }

        url = f"{BASE_URL}/v1/projects/{PROJECT_ID}/artifacts"
        res = requests.post(url, headers=headers, json=payload)
        
        if res.status_code in (200, 201):
            art = res.json()
            print(f"  ✅ Indexed '{rel_path}' -> ID: {art.get('id')}")
        else:
            print(f"  ❌ Failed to upload '{rel_path}': {res.status_code} - {res.text}")

print("✨ Done uploading sample memories!")
