import json
import time
import os
import requests

CONFIG_PATH = os.path.join(os.path.dirname(__file__), "..", "backend", "secrets", "config.json")
try:
    with open(CONFIG_PATH, "r") as f:
        config = json.load(f)
        API_KEY = "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"#config.get("openrouter_api_key", "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")
except Exception as e:
    print(f"Error reading config: {e}")
    API_KEY = ""

MODELS = [
    {"name": "DeepSeek V3", "id": "deepseek/deepseek-chat", "cost_in": 0.14, "cost_out": 0.28},
    {"name": "inclusionai/ling-2.6-flash", "id": "inclusionai/ling-2.6-flash", "cost_in": 0.01, "cost_out": 0.03},
    {"name": "Qwen 2.5 72B", "id": "qwen/qwen-2.5-72b-instruct", "cost_in": 0.35, "cost_out": 0.40},
]

TEST_CASES = [
    {
        "id": "TC-1",
        "name": "Pronoun Reference Resolution",
        "system_prompt": "You are a CaaS Reference Resolver. Extract pronouns (e.g. 'it', 'that') and map them to their ground-truth entity names from the context. Output ONLY valid JSON, nothing else: {\"references\": {\"it\": \"<entity>\"}, \"entities\": [{\"name\": \"<name>\", \"type\": \"<type>\"}]}",
        "messages": [
            {"role": "user", "content": "I am interested in staying at Grand Hyatt Bali for 3 nights."},
            {"role": "assistant", "content": "Grand Hyatt Bali is a luxury 5-star beachfront resort."},
            {"role": "user", "content": "How much does it cost per night?"}
        ],
        "expected_target": "Grand Hyatt Bali"
    },
    {
        "id": "TC-2",
        "name": "Ambiguous Demonstrative Phrase",
        "system_prompt": "Extract references. Map phrases like 'the second one' or 'option A' to exact values from context. Output ONLY valid JSON, nothing else: {\"references\": {\"<phrase>\": \"<resolved_target>\"}}",
        "messages": [
            {"role": "user", "content": "What are my flight options to Tokyo?"},
            {"role": "assistant", "content": "Option 1: Japan Airlines for $850. Option 2: All Nippon Airways for $790."},
            {"role": "user", "content": "I want to book the second one."}
        ],
        "expected_target": "option 2"
    },
    {
        "id": "TC-3",
        "name": "State & Topic Extraction",
        "system_prompt": "Extract the current active_topic and structured state variables (e.g. destination, budget). Output ONLY valid JSON, nothing else: {\"active_topic\": \"...\", \"state\": {\"key\": \"val\"}}",
        "messages": [
            {"role": "user", "content": "Planning a trip to Tokyo."},
            {"role": "assistant", "content": "What is your daily budget?"},
            {"role": "user", "content": "My max budget is $300 per day."}
        ],
        "expected_target": "tokyo"
    }
]

def run_test(model_id, system_prompt, messages):
    url = "https://openrouter.ai/api/v1/chat/completions"
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }
    
    # Send payload without enforcing rigid provider response_format if rejected
    payload = {
        "model": model_id,
        "messages": [{"role": "system", "content": system_prompt}] + messages,
        "temperature": 0.1,
    }
    
    start_t = time.perf_counter()
    try:
        res = requests.post(url, headers=headers, json=payload, timeout=20)
        elapsed_ms = (time.perf_counter() - start_t) * 1000
        
        if res.status_code != 200:
            return False, elapsed_ms, f"HTTP {res.status_code}: {res.text}", 0, 0
            
        data = res.json()
        content = data['choices'][0]['message']['content'].strip()
        
        # Clean markdown wrappers if present
        if content.startswith("```json"):
            content = content[7:]
        if content.startswith("```"):
            content = content[3:]
        if content.endswith("```"):
            content = content[:-3]
        content = content.strip()
        
        usage = data.get('usage', {})
        in_tokens = usage.get('prompt_tokens', 0)
        out_tokens = usage.get('completion_tokens', 0)
        
        # Validate JSON strictness
        parsed = json.loads(content)
        return True, elapsed_ms, parsed, in_tokens, out_tokens
    except Exception as e:
        elapsed_ms = (time.perf_counter() - start_t) * 1000
        return False, elapsed_ms, str(e), 0, 0

def benchmark():
    print("\n" + "="*85)
    print(" 🚀 OPENROUTER MODEL BENCHMARK: DEEPSEEK V3 vs MISTRAL SMALL vs QWEN 2.5")
    print("="*85 + "\n")
    
    results = {m["name"]: {"latency": [], "success": 0, "total_cost": 0.0, "accuracy": 0} for m in MODELS}
    
    for tc in TEST_CASES:
        print(f"📌 Running Test Case [{tc['id']}]: {tc['name']}...")
        
        for model in MODELS:
            name = model["name"]
            model_id = model["id"]
            
            ok, ms, output, in_t, out_t = run_test(model_id, tc["system_prompt"], tc["messages"])
            cost = (in_t / 1_000_000 * model["cost_in"]) + (out_t / 1_000_000 * model["cost_out"])
            
            # Check accuracy
            out_str = json.dumps(output).lower()
            acc = 1 if (ok and tc["expected_target"].lower() in out_str) else 0
            
            results[name]["latency"].append(ms)
            if ok:
                results[name]["success"] += 1
            results[name]["accuracy"] += acc
            results[name]["total_cost"] += cost
            
            status = "✅ PASS" if ok and acc == 1 else "❌ FAIL"
            print(f"  └─ [{name:<18}]: {status} | Latency: {ms:6.1f}ms | Cost: ${cost:.6f} | Output: {json.dumps(output)[:50]}...")
        print()

    # Print Final Summary Comparison Matrix
    print("="*85)
    print(" 📊 FINAL BENCHMARK SUMMARY & MODEL JUDGMENT")
    print("="*85)
    print(f"{'Model Name':<20} | {'Avg Latency':<12} | {'JSON Valid':<10} | {'Accuracy':<10} | {'Est 1k-Turns Cost':<18}")
    print("-" * 85)
    
    best_model = None
    best_score = -999
    
    for model in MODELS:
        name = model["name"]
        avg_ms = sum(results[name]["latency"]) / len(TEST_CASES) if TEST_CASES else 0
        valid_cnt = f"{results[name]['success']}/{len(TEST_CASES)}"
        acc_cnt = f"{results[name]['accuracy']}/{len(TEST_CASES)}"
        est_1k_cost = results[name]["total_cost"] * (1000 / len(TEST_CASES))
        
        print(f"{name:<20} | {avg_ms:>8.1f} ms | {valid_cnt:>10} | {acc_cnt:>10} | ${est_1k_cost:>16.4f}")
        
        if results[name]['success'] > 0:
            acc_score = results[name]['accuracy'] / len(TEST_CASES)
            speed_score = max(0, (3000 - avg_ms) / 3000)
            cost_score = max(0, (0.05 - est_1k_cost) / 0.05)
            composite = (acc_score * 0.5) + (speed_score * 0.3) + (cost_score * 0.2)
            
            if composite > best_score:
                best_score = composite
                best_model = name

    print("-" * 85)
    print(f"🏆 WINNER (Best Value for Money): {best_model}\n")

if __name__ == "__main__":
    benchmark()
