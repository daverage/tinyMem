"""Lightweight HTTP stub for LLM/embedding endpoints used in tests."""

import json
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer
from socketserver import ThreadingMixIn


class ThreadedHTTPServer(ThreadingMixIn, HTTPServer):
    daemon_threads = True


class StubLLMServer:
    """HTTP stub that can answer chat completions and embedding requests."""

    def __init__(self, chat_response=None, embedding_response=None):
        self.chat_response = chat_response or self._default_chat_response
        self.embedding_response = embedding_response or self._default_embedding_response
        self.requests = []

        server_self = self

        class Handler(BaseHTTPRequestHandler):
            def do_POST(self_inner):
                length = int(self_inner.headers.get("Content-Length", "0"))
                body = (self_inner.rfile.read(length).decode("utf-8")
                        if length > 0 else "")
                server_self.requests.append((self_inner.path, body))

                if self_inner.path.endswith("/chat/completions"):
                    response_body = server_self._render_chat_response(self_inner.path, body)
                elif self_inner.path.endswith("/embeddings"):
                    response_body = server_self._render_embedding_response(self_inner.path, body)
                else:
                    response_body = {"error": "unhandled path"}

                payload = (response_body if isinstance(response_body, str)
                           else json.dumps(response_body))

                self_inner.send_response(200)
                self_inner.send_header("Content-Type", "application/json")
                self_inner.send_header("Content-Length", str(len(payload.encode("utf-8"))))
                self_inner.end_headers()
                self_inner.wfile.write(payload.encode("utf-8"))

            def log_message(self_inner, format, *args):
                return

        self._httpd = ThreadedHTTPServer(("127.0.0.1", 0), Handler)
        self.port = self._httpd.server_address[1]
        self.base_url = f"http://127.0.0.1:{self.port}"
        self._thread = None

    def _render_chat_response(self, path, body):
        if callable(self.chat_response):
            return self.chat_response(path, body)
        return self.chat_response

    def _render_embedding_response(self, path, body):
        if callable(self.embedding_response):
            return self.embedding_response(path, body)
        return self.embedding_response

    @staticmethod
    def _default_chat_response(path, body):
        return {
            "choices": [
                {"message": {"content": "{\"status\": \"ok\"}"}}
            ]
        }

    @staticmethod
    def _default_embedding_response(path, body):
        return {"data": [{"embedding": [0.1, 0.2, 0.3]}]}

    def start(self):
        self._thread = threading.Thread(target=self._httpd.serve_forever, daemon=True)
        self._thread.start()

    def stop(self):
        self._httpd.shutdown()
        if self._thread is not None:
            self._thread.join(timeout=1)
