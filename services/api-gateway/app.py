import logging
import os
import time
import uuid
from datetime import datetime, timezone

from flask import Flask, jsonify, request
from flask_cors import CORS

app = Flask(__name__)
CORS(app)

logging.basicConfig(
    level=os.getenv("LOG_LEVEL", "INFO"),
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("api-gateway")

endpoints_store: dict[str, dict] = {}

START_TIME = time.time()


@app.route("/health", methods=["GET"])
def health():
    return jsonify({
        "status": "healthy",
        "service": "api-gateway",
        "uptime_seconds": round(time.time() - START_TIME, 2),
        "timestamp": datetime.now(timezone.utc).isoformat(),
    })


@app.route("/api/v1/endpoints", methods=["GET"])
def list_endpoints():
    logger.info("Listing all monitored endpoints")
    return jsonify({"endpoints": list(endpoints_store.values())})


@app.route("/api/v1/endpoints", methods=["POST"])
def create_endpoint():
    data = request.get_json()
    if not data or "url" not in data:
        logger.warning("Create endpoint called without url")
        return jsonify({"error": "url is required"}), 400

    url = data["url"]
    name = data.get("name", url)
    interval = data.get("interval_seconds", 30)

    if not isinstance(interval, (int, float)) or interval < 5:
        return jsonify({"error": "interval_seconds must be >= 5"}), 400

    endpoint_id = str(uuid.uuid4())
    endpoint = {
        "id": endpoint_id,
        "url": url,
        "name": name,
        "interval_seconds": interval,
        "created_at": datetime.now(timezone.utc).isoformat(),
        "status": "pending",
    }
    endpoints_store[endpoint_id] = endpoint
    logger.info("Created endpoint %s for url %s", endpoint_id, url)
    return jsonify(endpoint), 201


@app.route("/api/v1/endpoints/<endpoint_id>", methods=["GET"])
def get_endpoint(endpoint_id):
    endpoint = endpoints_store.get(endpoint_id)
    if not endpoint:
        return jsonify({"error": "endpoint not found"}), 404
    return jsonify(endpoint)


@app.route("/api/v1/endpoints/<endpoint_id>", methods=["DELETE"])
def delete_endpoint(endpoint_id):
    if endpoint_id not in endpoints_store:
        return jsonify({"error": "endpoint not found"}), 404
    del endpoints_store[endpoint_id]
    logger.info("Deleted endpoint %s", endpoint_id)
    return jsonify({"message": "deleted"}), 200


@app.route("/api/v1/endpoints/<endpoint_id>/status", methods=["PUT"])
def update_status(endpoint_id):
    endpoint = endpoints_store.get(endpoint_id)
    if not endpoint:
        return jsonify({"error": "endpoint not found"}), 404

    data = request.get_json()
    if not data or "status" not in data:
        return jsonify({"error": "status is required"}), 400

    allowed = {"healthy", "unhealthy", "pending", "timeout"}
    if data["status"] not in allowed:
        return jsonify({"error": f"status must be one of {allowed}"}), 400

    endpoint["status"] = data["status"]
    endpoint["last_checked"] = datetime.now(timezone.utc).isoformat()
    if "response_time_ms" in data:
        endpoint["response_time_ms"] = data["response_time_ms"]

    logger.info("Updated endpoint %s status to %s", endpoint_id, data["status"])
    return jsonify(endpoint)


def create_app():
    return app


if __name__ == "__main__":
    port = int(os.getenv("API_PORT", "8001"))
    debug = os.getenv("FLASK_DEBUG", "false").lower() == "true"
    logger.info("Starting API Gateway on port %d", port)
    app.run(host="0.0.0.0", port=port, debug=debug)
