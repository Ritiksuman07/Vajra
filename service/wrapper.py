#!/usr/bin/env python3
"""Windows Service Wrapper for Vajra."""
import sys, os, subprocess, signal, time
sys.path.insert(0, os.path.dirname(os.path.dirname(__file__)))

SERVICE_NAME = "Vajra"
SERVICE_EXE = os.path.join(os.path.dirname(__file__), "..", "dist", "vajra-service.exe")

class VajraService:
    def __init__(self):
        self.process = None
        self.running = False

    def start(self):
        print(f"Starting {SERVICE_NAME}...")
        if not os.path.exists(SERVICE_EXE):
            print(f"Error: {SERVICE_EXE} not found")
            return False
        self.process = subprocess.Popen([SERVICE_EXE], stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        self.running = True
        print(f"{SERVICE_NAME} started (PID: {self.process.pid})")
        return True

    def stop(self):
        print(f"Stopping {SERVICE_NAME}...")
        if self.process:
            self.process.terminate()
            try:
                self.process.wait(timeout=5)
            except:
                self.process.kill()
            self.process = None
        self.running = False
        print(f"{SERVICE_NAME} stopped")
        return True

if __name__ == "__main__":
    svc = VajraService()
    if len(sys.argv) < 2:
        print("Usage: wrapper [start|stop]")
    else:
        cmd = sys.argv[1]
        if cmd == "start": svc.start()
        elif cmd == "stop": svc.stop()
