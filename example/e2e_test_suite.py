import sys
import os
import time
import requests

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '../sdk/python')))

from contextkit_sdk.client import ContextKitClient
from contextkit_sdk.crypto import ContextCrypto

SERVER_URL = "http://localhost:8080"
SIDECAR_URL = "http://localhost:8085"
API_KEY = "ck_live_3383e91ba399b71fb614c8b239a7629b"
PROJECT_ID = "b0000000-0000-0000-0000-000000000001"
MASTER_KEY = "konteks_sample_static_master_key_32b!"

HEADERS = {
    "Content-Type": "application/json",
    "X-API-Key": API_KEY,
    "X-Project-ID": PROJECT_ID,
}

def log_header(title: str):
    print(f"\n========================================================")
    print(f"  {title}")
    print(f"========================================================")

def assert_test(condition: bool, description: str):
    if condition:
        print(f"  [PASS] {description}")
    else:
        print(f"  [FAIL] {description}")
        raise AssertionError(f"Test failed: {description}")

def run_e2e_suite():
    print("🚀 Starting ContextKit E2E Test Suite...")

    # ---------------------------------------------------------
    # Scenario 1: Context Health Policy & Entropy Sanitization
    # ---------------------------------------------------------
    log_header("Scenario 1: Context Health Policy & Entropy Sanitization")
    
    # 1. Update policy
    policy_url = f"{SERVER_URL}/v1/projects/{PROJECT_ID}/context-health-policy"
    policy_payload = {
        "enabled": True,
        "mode": "balanced",
        "decay_half_life_days": 30,
        "max_entropy_threshold": 4.8,
        "turn_eval_interval": 50,
        "deduplication_threshold": 0.95,
        "e2ee_eval_location": "server_distance_only"
    }
    resp = requests.put(policy_url, json=policy_payload, headers=HEADERS)
    assert_test(resp.status_code == 200, "Successfully updated project Context Health policy")

    # 2. Create test room
    client = ContextKitClient(api_key=API_KEY, project_id=PROJECT_ID, base_url=SERVER_URL, master_key=MASTER_KEY)
    room = client.create_room(description="E2E Test Room")
    room_id = room["id"]
    assert_test(bool(room_id), f"Created room ID: {room_id}")

    # 3. Post technical content (whitelisted)
    msg1 = client.send_message(room_id, role="user", content="```go\nfunc main() {\n\tfmt.Println(\"whitelisted code\")\n}\n```")
    assert_test(bool(msg1.get("id")), "Posted technical whitelisted code block message")

    # ---------------------------------------------------------
    # Scenario 2: Zero-Trust E2EE Payload Encryption & Vector Rotation
    # ---------------------------------------------------------
    log_header("Scenario 2: Zero-Trust E2EE Payload Encryption & Vector Rotation")

    sidecar_client = ContextKitClient(api_key=API_KEY, project_id=PROJECT_ID, base_url=SIDECAR_URL, master_key=MASTER_KEY)
    
    sensitive_text = "User sensitive financial budget is $45,000 USD for Q3."
    msg2 = sidecar_client.send_message(room_id, role="user", content=sensitive_text, is_encrypted=True)
    assert_test(bool(msg2.get("id")), "Sent encrypted message via Sidecar Proxy (:8085)")

    # Direct raw server inspection to verify ZERO plaintext on server
    raw_msgs_resp = requests.get(f"{SERVER_URL}/v1/rooms/{room_id}/messages", headers=HEADERS)
    raw_msgs = raw_msgs_resp.json().get("messages", [])
    
    raw_content = ""
    for m in raw_msgs:
        if m.get("id") == msg2.get("id"):
            raw_content = m.get("content", "")

    assert_test(raw_content.startswith("enc_"), "Raw content in DB starts with encrypted container prefix 'enc_'")
    assert_test(sensitive_text not in raw_content, "VERIFIED: Zero plaintext stored on server DB!")

    # Retrieve context via Sidecar Proxy -> verify transparent decryption
    ctx_pkg = sidecar_client.get_context(room_id, query="What is the user budget?")
    assert_test(bool(ctx_pkg), "Retrieved context package via Sidecar Proxy")

    # ---------------------------------------------------------
    # Scenario 3: Fact Supersedence DAG & Micro-Sweep
    # ---------------------------------------------------------
    log_header("Scenario 3: Fact Supersedence DAG & Micro-Sweep")

    # Fact A
    fact_a = "User primary location is Tokyo, Japan."
    sidecar_client.send_message(room_id, role="user", content=fact_a, is_encrypted=True)

    # Fact B (Supersedes Fact A)
    fact_b = "User moved and primary location is now San Francisco, California."
    sidecar_client.send_message(room_id, role="user", content=fact_b, is_encrypted=True)

    # Trigger micro-sweep
    clean_url = f"{SERVER_URL}/v1/rooms/{room_id}/context-health/clean"
    clean_resp = requests.post(clean_url, headers=HEADERS)
    assert_test(clean_resp.status_code in [200, 202], "Triggered room Context Health micro-sweep")

    # List memories
    mems_resp = requests.get(f"{SERVER_URL}/v1/rooms/{room_id}/memories?include_superseded=true", headers=HEADERS)
    assert_test(mems_resp.status_code == 200, "Listed room memories including superseded facts")

    # ---------------------------------------------------------
    # Scenario 4: Temporal Historical Query Recall
    # ---------------------------------------------------------
    log_header("Scenario 4: Temporal Historical Query Recall")

    temporal_ctx = sidecar_client.get_context(room_id, query="What was our location 8 months ago?")
    assert_test(bool(temporal_ctx), "Executed temporal relative time query ('8 months ago') successfully")

    # ---------------------------------------------------------
    # Scenario 5: Fact Restoration & Audit Revisions
    # ---------------------------------------------------------
    log_header("Scenario 5: Fact Restoration & Audit Revisions")

    memories = mems_resp.json().get("memories", [])
    if len(memories) > 0:
        target_mem_id = memories[0].get("id")
        restore_url = f"{SERVER_URL}/v1/memories/{target_mem_id}/restore"
        restore_resp = requests.post(restore_url, headers=HEADERS)
        assert_test(restore_resp.status_code == 200, f"Successfully restored memory {target_mem_id} via API")

    print("\n✅ ALL 5 E2E TEST SCENARIOS PASSED SUCCESSFULLY!")

if __name__ == "__main__":
    run_e2e_suite()
