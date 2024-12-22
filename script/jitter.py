import http.server
import random
import time

class JitteryHTTPRequestHandler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        # Add jitter/latency before responding
        max = random.uniform(0.1, 10.0)
        jitter = random.uniform(0.1, max)  # Random delay between 100ms and 1s
        time.sleep(jitter)

        # Proceed with the default GET handling
        super().do_GET()

    def do_POST(self):
        # Add jitter/latency before responding
        jitter = random.uniform(0.1, 10.0)  # Random delay between 100ms and 1s
        time.sleep(jitter)

        # Proceed with the default POST handling
        super().do_POST()

# Start the server
if __name__ == "__main__":
    port = 8080
    with http.server.ThreadingHTTPServer(("0.0.0.0", port), JitteryHTTPRequestHandler) as httpd:
        print(f"Serving with latency on port {port}")
        httpd.serve_forever()