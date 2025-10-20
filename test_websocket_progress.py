#!/usr/bin/env python3
"""
Test script to download a YouTube video and monitor WebSocket progress updates
"""

import asyncio
import websockets
import json
import requests
import sys

BACKEND_URL = "http://localhost:8080"
WS_URL = "ws://localhost:8080/ws"
TEST_VIDEO_URL = "https://youtu.be/xR-40NwDI7U?si=EiBsbE15pOIyOuKf"

async def monitor_websocket():
    """Connect to WebSocket and monitor messages"""
    print(f"Connecting to WebSocket: {WS_URL}")
    
    try:
        async with websockets.connect(WS_URL) as websocket:
            print("✅ WebSocket connected!")
            
            # Subscribe to all downloads
            subscribe_msg = {
                "action": "subscribeAll"
            }
            await websocket.send(json.dumps(subscribe_msg))
            print("📡 Subscribed to all downloads")
            
            # Listen for messages
            message_count = 0
            progress_updates = []
            
            async for message in websocket:
                message_count += 1
                try:
                    data = json.loads(message)
                    msg_type = data.get('type', 'unknown')
                    
                    print(f"\n[Message #{message_count}] Type: {msg_type}")
                    
                    if msg_type == 'queueStatus':
                        queue = data.get('queue', [])
                        print(f"  Queue size: {len(queue)}")
                        for job in queue:
                            job_info = job.get('job', {})
                            status = job.get('status', 'unknown')
                            progress = job.get('progress', 'N/A')
                            print(f"  - Job {job_info.get('id', 'N/A')}: {status}")
                            print(f"    Progress: {progress}")
                    
                    elif msg_type == 'progress':
                        download_id = data.get('downloadId', 'N/A')
                        message_text = data.get('message', 'N/A')
                        percent = data.get('percent', 0)
                        
                        progress_updates.append({
                            'downloadId': download_id,
                            'message': message_text,
                            'percent': percent
                        })
                        
                        print(f"  📥 Download ID: {download_id}")
                        print(f"  📊 Progress: {percent:.2f}%")
                        print(f"  💬 Message: {message_text}")
                        
                    elif msg_type == 'done':
                        download_id = data.get('downloadId', 'N/A')
                        message_text = data.get('message', 'N/A')
                        file_name = data.get('file', 'N/A')
                        
                        print(f"  ✅ Download ID: {download_id}")
                        print(f"  📁 File: {file_name}")
                        print(f"  💬 Message: {message_text}")
                        
                        # Print summary
                        print(f"\n{'='*60}")
                        print(f"DOWNLOAD COMPLETE!")
                        print(f"Total messages received: {message_count}")
                        print(f"Progress updates: {len(progress_updates)}")
                        if progress_updates:
                            print(f"First update: {progress_updates[0]['percent']:.2f}%")
                            print(f"Last update: {progress_updates[-1]['percent']:.2f}%")
                        print(f"{'='*60}")
                        
                        # Exit after download completes
                        break
                        
                    elif msg_type == 'error':
                        download_id = data.get('downloadId', 'N/A')
                        message_text = data.get('message', 'N/A')
                        
                        print(f"  ❌ Download ID: {download_id}")
                        print(f"  💬 Error: {message_text}")
                        break
                        
                    elif msg_type == 'list':
                        videos = data.get('videos', [])
                        print(f"  📂 Videos available: {len(videos)}")
                        
                    else:
                        print(f"  Raw data: {json.dumps(data, indent=2)}")
                        
                except json.JSONDecodeError as e:
                    print(f"  ⚠️  Failed to parse message: {e}")
                    print(f"  Raw message: {message}")
                    
    except websockets.exceptions.WebSocketException as e:
        print(f"❌ WebSocket error: {e}")
        return False
    except Exception as e:
        print(f"❌ Unexpected error: {e}")
        return False
    
    return True

def trigger_download():
    """Trigger a YouTube download via HTTP API"""
    print(f"\n{'='*60}")
    print(f"Triggering download: {TEST_VIDEO_URL}")
    print(f"{'='*60}\n")
    
    try:
        response = requests.post(
            f"{BACKEND_URL}/urlyt",
            json={
                "url": TEST_VIDEO_URL,
                "mode": "download",
                "autoPlay": False
            },
            timeout=10
        )
        
        if response.status_code == 200:
            data = response.json()
            print(f"✅ Download triggered successfully!")
            print(f"   Response: {json.dumps(data, indent=2)}")
            return data.get('file', None)
        else:
            print(f"❌ Failed to trigger download: {response.status_code}")
            print(f"   Response: {response.text}")
            return None
            
    except requests.exceptions.RequestException as e:
        print(f"❌ HTTP request failed: {e}")
        return None

async def main():
    """Main test function"""
    print("="*60)
    print("YouTube Download WebSocket Progress Test")
    print("="*60)
    
    # Start WebSocket monitoring in background
    ws_task = asyncio.create_task(monitor_websocket())
    
    # Wait a bit for WebSocket to connect
    await asyncio.sleep(1)
    
    # Trigger the download
    download_id = trigger_download()
    
    if not download_id:
        print("❌ Failed to trigger download, cancelling test")
        ws_task.cancel()
        return 1
    
    print(f"\n⏳ Monitoring WebSocket for progress updates...")
    print(f"   (This will take a few seconds while the video downloads)\n")
    
    # Wait for WebSocket task to complete
    try:
        result = await ws_task
        if result:
            print("\n✅ Test completed successfully!")
            return 0
        else:
            print("\n❌ Test failed!")
            return 1
    except asyncio.CancelledError:
        print("\n⚠️  Test cancelled")
        return 1

if __name__ == "__main__":
    try:
        exit_code = asyncio.run(main())
        sys.exit(exit_code)
    except KeyboardInterrupt:
        print("\n\n⚠️  Test interrupted by user")
        sys.exit(1)
