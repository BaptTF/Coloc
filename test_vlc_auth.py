#!/usr/bin/env python3

import requests

# Test our Go server's VLC authentication proxy
SERVER_URL = "http://localhost:8080"
VLC_URL = "http://192.168.4.29:8080"  # Change this to your VLC URL

def test_vlc_auth():
    print("Testing VLC authentication via Go server...")
    
    # Step 1: Get challenge via our proxy
    print("1. Getting challenge via Go proxy...")
    try:
        resp = requests.get(f"{SERVER_URL}/vlc/code?vlc={VLC_URL}")
        if resp.status_code != 200:
            print(f"❌ Failed to get challenge: {resp.status_code}")
            print(f"Response: {resp.text}")
            return False
        
        challenge = resp.text
        print(f"✅ Got challenge: '{challenge}' (length: {len(challenge)})")
    except Exception as e:
        print(f"❌ Error getting challenge: {e}")
        return False
    
    # Step 2: Get user input for code
    code = input("Enter the 4-digit code displayed on VLC: ")
    
    # Step 3: Send code to our proxy (raw code, server will hash it)
    print("3. Sending raw code to Go proxy...")
    try:
        resp = requests.post(
            f"{SERVER_URL}/vlc/verify-code?vlc={VLC_URL}",
            headers={"Content-Type": "application/json"},
            json={"code": code}
        )
        
        print(f"Status: {resp.status_code}")
        print(f"Response: {resp.text}")
        
        if resp.status_code == 200:
            print("✅ Authentication successful!")
            return True
        else:
            print("❌ Authentication failed!")
            return False
            
    except Exception as e:
        print(f"❌ Error during verification: {e}")
        return False

if __name__ == "__main__":
    test_vlc_auth()
