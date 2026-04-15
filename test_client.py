#!/usr/bin/env python3
import requests
import json
import sys

BASE_URL = "http://localhost:8080"

def test_health():
    print("\n=== Testing Health Endpoint ===")
    resp = requests.get(f"{BASE_URL}/health")
    print(f"Status: {resp.status_code}")
    print(f"Response: {resp.json()}")

def test_models():
    print("\n=== Testing Models Endpoint ===")
    resp = requests.get(f"{BASE_URL}/v1/models")
    print(f"Status: {resp.status_code}")
    print(f"Response: {json.dumps(resp.json(), indent=2)}")

def test_chat_completion():
    print("\n=== Testing Chat Completion (Non-Streaming) ===")
    payload = {
        "model": "Qwen/Qwen2.5-7B-Instruct",
        "messages": [
            {"role": "user", "content": "你好，请用一句话介绍自己"}
        ],
        "max_tokens": 100
    }
    
    resp = requests.post(
        f"{BASE_URL}/v1/chat/completions",
        headers={"Content-Type": "application/json"},
        json=payload
    )
    
    print(f"Status: {resp.status_code}")
    if resp.status_code == 200:
        result = resp.json()
        print(f"Response: {json.dumps(result, indent=2, ensure_ascii=False)}")
        if result.get("choices"):
            print(f"\nAssistant: {result['choices'][0]['message']['content']}")
    else:
        print(f"Error: {resp.text}")

def test_chat_completion_stream():
    print("\n=== Testing Chat Completion (Streaming) ===")
    payload = {
        "model": "Qwen/Qwen2.5-7B-Instruct",
        "messages": [
            {"role": "user", "content": "数到10"}
        ],
        "stream": True,
        "max_tokens": 100
    }
    
    resp = requests.post(
        f"{BASE_URL}/v1/chat/completions",
        headers={"Content-Type": "application/json"},
        json=payload,
        stream=True
    )
    
    print(f"Status: {resp.status_code}")
    print("Streaming response:")
    
    full_content = ""
    for line in resp.iter_lines():
        if line:
            line = line.decode('utf-8')
            if line.startswith("data: "):
                data = line[6:]
                if data == "[DONE]":
                    print("\n[Stream completed]")
                    break
                
                try:
                    chunk = json.loads(data)
                    if chunk.get("choices"):
                        delta = chunk["choices"][0].get("delta", {})
                        content = delta.get("content", "")
                        if content:
                            print(content, end="", flush=True)
                            full_content += content
                except json.JSONDecodeError:
                    pass
    
    print(f"\n\nFull response: {full_content}")

def test_openai_sdk_compatibility():
    print("\n=== Testing OpenAI SDK Compatibility ===")
    try:
        from openai import OpenAI
        
        client = OpenAI(
            api_key="test-key",
            base_url=BASE_URL
        )
        
        print("Creating chat completion...")
        response = client.chat.completions.create(
            model="Qwen/Qwen2.5-7B-Instruct",
            messages=[
                {"role": "user", "content": "你好"}
            ],
            max_tokens=50
        )
        
        print(f"Response ID: {response.id}")
        print(f"Model: {response.model}")
        print(f"Content: {response.choices[0].message.content}")
        print("✓ OpenAI SDK compatibility test passed!")
        
    except ImportError:
        print("OpenAI SDK not installed. Install with: pip install openai")
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    print("LLM Gateway API Test Client")
    print("=" * 50)
    
    try:
        if len(sys.argv) > 1:
            test_name = sys.argv[1]
            if test_name == "health":
                test_health()
            elif test_name == "models":
                test_models()
            elif test_name == "chat":
                test_chat_completion()
            elif test_name == "stream":
                test_chat_completion_stream()
            elif test_name == "sdk":
                test_openai_sdk_compatibility()
            else:
                print(f"Unknown test: {test_name}")
        else:
            test_health()
            test_models()
            test_chat_completion()
            test_chat_completion_stream()
    except requests.exceptions.ConnectionError:
        print("\nError: Cannot connect to server. Make sure the server is running on http://localhost:8080")
    except Exception as e:
        print(f"\nError: {e}")
