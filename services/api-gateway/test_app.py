import pytest

from app import create_app


@pytest.fixture
def client():
    app = create_app()
    app.config["TESTING"] = True
    with app.test_client() as c:
        yield c


def test_health(client):
    resp = client.get("/health")
    assert resp.status_code == 200
    data = resp.get_json()
    assert data["status"] == "healthy"
    assert data["service"] == "api-gateway"
    assert "uptime_seconds" in data


def test_list_endpoints_empty(client):
    resp = client.get("/api/v1/endpoints")
    assert resp.status_code == 200
    assert resp.get_json()["endpoints"] == []


def test_create_endpoint(client):
    resp = client.post("/api/v1/endpoints", json={
        "url": "https://example.com",
        "name": "Example",
        "interval_seconds": 30,
    })
    assert resp.status_code == 201
    data = resp.get_json()
    assert data["url"] == "https://example.com"
    assert data["name"] == "Example"
    assert data["status"] == "pending"
    assert "id" in data


def test_create_endpoint_missing_url(client):
    resp = client.post("/api/v1/endpoints", json={"name": "NoURL"})
    assert resp.status_code == 400


def test_create_endpoint_invalid_interval(client):
    resp = client.post("/api/v1/endpoints", json={
        "url": "https://example.com",
        "interval_seconds": 2,
    })
    assert resp.status_code == 400


def test_get_endpoint(client):
    resp = client.post("/api/v1/endpoints", json={"url": "https://example.com"})
    eid = resp.get_json()["id"]
    resp = client.get(f"/api/v1/endpoints/{eid}")
    assert resp.status_code == 200
    assert resp.get_json()["id"] == eid


def test_get_endpoint_not_found(client):
    resp = client.get("/api/v1/endpoints/nonexistent")
    assert resp.status_code == 404


def test_delete_endpoint(client):
    resp = client.post("/api/v1/endpoints", json={"url": "https://example.com"})
    eid = resp.get_json()["id"]
    resp = client.delete(f"/api/v1/endpoints/{eid}")
    assert resp.status_code == 200
    resp = client.get(f"/api/v1/endpoints/{eid}")
    assert resp.status_code == 404


def test_delete_endpoint_not_found(client):
    resp = client.delete("/api/v1/endpoints/nonexistent")
    assert resp.status_code == 404


def test_update_status(client):
    resp = client.post("/api/v1/endpoints", json={"url": "https://example.com"})
    eid = resp.get_json()["id"]
    resp = client.put(f"/api/v1/endpoints/{eid}/status", json={
        "status": "healthy",
        "response_time_ms": 42,
    })
    assert resp.status_code == 200
    data = resp.get_json()
    assert data["status"] == "healthy"
    assert data["response_time_ms"] == 42
    assert "last_checked" in data


def test_update_status_invalid(client):
    resp = client.post("/api/v1/endpoints", json={"url": "https://example.com"})
    eid = resp.get_json()["id"]
    resp = client.put(f"/api/v1/endpoints/{eid}/status", json={"status": "bogus"})
    assert resp.status_code == 400


def test_update_status_not_found(client):
    resp = client.put("/api/v1/endpoints/nonexistent/status", json={"status": "healthy"})
    assert resp.status_code == 404


def test_update_status_missing_field(client):
    resp = client.post("/api/v1/endpoints", json={"url": "https://example.com"})
    eid = resp.get_json()["id"]
    resp = client.put(f"/api/v1/endpoints/{eid}/status", json={})
    assert resp.status_code == 400
