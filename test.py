import requests
from hashlib import sha256

SERVER_IP = "192.168.4.29"
SERVER_PORT = 8080

server_url = f"http://{SERVER_IP}:{SERVER_PORT}"

s = requests.Session()

resp = s.post(server_url + "/code", {"challenge": ""})

if resp.status_code != 200:
    raise Exception(f"Server responded with {resp.status_code}")

challenge = resp.text

code = input("Enter code : ")

answer = sha256(bytes(code + challenge, "utf-8")).hexdigest()

resp = s.post(server_url + "/verify-code", {"code": answer})

if "user_session" not in s.cookies:
    print("Authentication failed")
    exit(1)
print(resp)
print("Authentication sucessful")
print(f"user_session={s.cookies['user_session']}")