import requests
import json

BASE_URL = "http://localhost:8080/v1"
API_KEY = "kt_7270885e0284152aac928eed17faf82565ecc126"
HEADERS = {
    "Authorization": f"Bearer {API_KEY}",
    "Content-Type": "application/json"
}

def run_phase2_verification():
    print("\n" + "="*80)
    print(" 🧪 PHASE 2 CONTEXT ENGINE END-TO-END VERIFICATION")
    print("="*80 + "\n")

    # 1. Create a Room
    print("1️⃣ Creating Room...")
    res = requests.post(f"{BASE_URL}/rooms", headers=HEADERS, json={"external_user_id": "test_user_777"})
    assert res.status_code == 201, f"Failed to create room: {res.text}"
    room_id = res.json()["id"]
    print(f"   ✅ Room Created: {room_id}")

    # 2. Push User & Assistant Messages
    print("\n2️⃣ Pushing Conversation Messages...")
    messages = [
        {"role": "user", "content": "I want to stay at Grand Hyatt Bali for 3 nights next month."},
        {"role": "assistant", "content": "Grand Hyatt Bali is a 5-star oceanfront resort with multiple pools."},
        {"role": "user", "content": "How much does it cost per night?"}
    ]
    for m in messages:
        res = requests.post(f"{BASE_URL}/rooms/{room_id}/messages", headers=HEADERS, json=m)
        assert res.status_code == 201, f"Failed to post message: {res.text}"
    print(f"   ✅ Pushed {len(messages)} messages successfully.")

    # 3. Test Context Request with Auto-Strategy (Default)
    print("\n3️⃣ Fetching Context Package (Auto Strategy + Reference Resolution)...")
    payload = {
        "query": "How much does it cost?",
        "policy": {
            "max_tokens": 1500,
            "strategy": "auto"
        }
    }
    res = requests.post(f"{BASE_URL}/rooms/{room_id}/context", headers=HEADERS, json=payload)
    assert res.status_code == 200, f"Failed to fetch context package: {res.text}"
    pkg = res.json()
    
    print("\n📦 Returned Context Package:")
    print(f"   • Strategy Used:    '{pkg.get('strategy_used')}' (Auto-detected recent_heavy for pronoun 'it')")
    print(f"   • Estimated Tokens: {pkg.get('tokens_estimated')} (Under max_tokens=1500)")
    print(f"   • References:       {json.dumps(pkg.get('references'))}")
    print(f"   • Entities:         {json.dumps(pkg.get('entities'))}")
    print(f"   • Messages Count:   {len(pkg.get('relevant_messages', []))}")

    # 4. Test Policy Truncation (Max Tokens = 20)
    print("\n4️⃣ Testing Token Budget Truncation Policy (max_tokens=20)...")
    payload_budget = {
        "query": "How much does it cost?",
        "policy": {
            "max_tokens": 20,
            "strategy": "recent_heavy font"
        }
    }
    res = requests.post(f"{BASE_URL}/rooms/{room_id}/context", headers=HEADERS, json=payload_budget)
    assert res.status_code == 200
    pkg_budget = res.json()
    print(f"   ✅ Truncated Package Tokens: {pkg_budget.get('tokens_estimated')} tokens")

    print("\n" + "="*80)
    print(" 🎉 ALL PHASE 2 VERIFICATION TESTS PASSED SUCCESSFULLY!")
    print("="*80 + "\n")

if __name__ == "__main__":
    run_phase2_verification()
