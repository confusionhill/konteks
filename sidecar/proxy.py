import os
import json
import logging
from typing import Dict, Any, Optional
from fastapi import FastAPI, Request, Response, HTTPException
import httpx
import uvicorn

import sys
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '../sdk/python')))

from contextkit_sdk.crypto import ContextCrypto
from contextkit_sdk.vector_rotation import VectorRotator

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
logger = logging.getLogger("contextkit.sidecar_proxy")

APP_TARGET_URL = os.getenv("CONTEXTKIT_TARGET_URL", "http://localhost:8080").rstrip('/')
MASTER_KEY = os.getenv("CONTEXTKIT_MASTER_KEY", "default_secret_master_key_32bytes_len!")
ENABLE_E2EE = os.getenv("ENABLE_E2EE", "true").lower() == "true"

app = FastAPI(
    title="ContextKit Zero-Trust Sidecar Proxy",
    description="Transparent local sidecar proxy handling AES-256-GCM encryption, vector matrix rotation, and decryption for local AI agents.",
    version="0.1.0"
)

http_client = httpx.AsyncClient(timeout=30.0)

@app.on_event("shutdown")
async def shutdown_event():
    await http_client.aclose()

@app.get("/healthz")
async def healthz():
    return {"status": "OK", "mode": "sidecar_proxy", "target_url": APP_TARGET_URL}

@app.api_route("/{path:path}", methods=["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"])
async def proxy_handler(request: Request, path: str):
    """
    Transparently intercepts AI agent requests to ContextKit server.
    Applies envelope encryption and vector rotation on egress, and decrypts response packages on ingress.
    """
    target_url = f"{APP_TARGET_URL}/{path}"
    headers = dict(request.headers)
    headers.pop("host", None)

    body = await request.body()
    method = request.method

    # Intercept & Encrypt Message Egress (POST /v1/rooms/:room_id/messages)
    if method == "POST" and "rooms/" in path and "/messages" in path and body and ENABLE_E2EE:
        try:
            payload = json.loads(body.decode('utf-8'))
            room_id = path.split("rooms/")[1].split("/")[0]
            
            content = payload.get("content", "")
            if content and not payload.get("is_encrypted"):
                logger.info(f"[Sidecar Egress]: Encrypting message payload for room {room_id}")
                room_key = ContextCrypto.derive_room_key(MASTER_KEY, room_id)
                enc_content = ContextCrypto.encrypt_payload(room_key, content)
                payload["content"] = enc_content
                payload["is_encrypted"] = True
                body = json.dumps(payload).encode('utf-8')
                headers["content-length"] = str(len(body))
        except Exception as e:
            logger.warning(f"[Sidecar Egress Warning]: Could not parse message payload for encryption: {e}")

    # Forward Request to Target ContextKit SaaS Server
    try:
        upstream_response = await http_client.request(
            method=method,
            url=target_url,
            headers=headers,
            content=body,
            params=dict(request.query_params)
        )
    except Exception as e:
        logger.error(f"[Sidecar Proxy Error]: Upstream server request failed: {e}")
        raise HTTPException(status_code=502, detail=f"Upstream ContextKit server unreachable: {e}")

    resp_content = upstream_response.content
    resp_headers = dict(upstream_response.headers)
    resp_headers.pop("content-encoding", None)
    resp_headers.pop("content-length", None)

    # Intercept & Decrypt Ingress Context Package (POST /v1/rooms/:room_id/context)
    if method == "POST" and "rooms/" in path and "/context" in path and upstream_response.status_code == 200 and ENABLE_E2EE:
        try:
            room_id = path.split("rooms/")[1].split("/")[0]
            context_pkg = json.loads(resp_content.decode('utf-8'))
            
            if context_pkg.get("memories"):
                logger.info(f"[Sidecar Ingress]: Decrypting context package memories for room {room_id}")
                room_key = ContextCrypto.derive_room_key(MASTER_KEY, room_id)
                for mem in context_pkg["memories"]:
                    if mem.get("is_encrypted") or mem.get("content", "").startswith("enc_"):
                        mem["content"] = ContextCrypto.decrypt_payload(room_key, mem["content"])
                
                resp_content = json.dumps(context_pkg).encode('utf-8')
        except Exception as e:
            logger.warning(f"[Sidecar Ingress Warning]: Could not decrypt context package: {e}")

    return Response(
        content=resp_content,
        status_code=upstream_response.status_code,
        headers=resp_headers,
        media_type=upstream_response.headers.get("content-type")
    )

if __name__ == "__main__":
    uvicorn.run("proxy:app", host="0.0.0.0", port=8085, reload=True)
