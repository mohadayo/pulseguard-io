import request from "supertest";
import { app, eventsStore } from "./app";

beforeEach(() => {
  eventsStore.length = 0;
});

describe("GET /health", () => {
  it("returns healthy status", async () => {
    const res = await request(app).get("/health");
    expect(res.status).toBe(200);
    expect(res.body.status).toBe("healthy");
    expect(res.body.service).toBe("analytics-dashboard");
    expect(res.body).toHaveProperty("uptime_seconds");
  });
});

describe("POST /api/v1/events", () => {
  it("creates a new event", async () => {
    const res = await request(app).post("/api/v1/events").send({
      endpointUrl: "https://example.com",
      status: "healthy",
      statusCode: 200,
      responseTimeMs: 42,
    });
    expect(res.status).toBe(201);
    expect(res.body.endpointUrl).toBe("https://example.com");
    expect(res.body.status).toBe("healthy");
    expect(res.body.id).toMatch(/^evt-/);
  });

  it("rejects missing endpointUrl", async () => {
    const res = await request(app).post("/api/v1/events").send({
      status: "healthy",
      responseTimeMs: 10,
    });
    expect(res.status).toBe(400);
  });

  it("rejects missing status", async () => {
    const res = await request(app).post("/api/v1/events").send({
      endpointUrl: "https://example.com",
      responseTimeMs: 10,
    });
    expect(res.status).toBe(400);
  });

  it("rejects negative responseTimeMs", async () => {
    const res = await request(app).post("/api/v1/events").send({
      endpointUrl: "https://example.com",
      status: "healthy",
      responseTimeMs: -1,
    });
    expect(res.status).toBe(400);
  });
});

describe("GET /api/v1/events", () => {
  it("returns empty list initially", async () => {
    const res = await request(app).get("/api/v1/events");
    expect(res.status).toBe(200);
    expect(res.body.events).toEqual([]);
    expect(res.body.total).toBe(0);
  });

  it("returns events after posting", async () => {
    await request(app).post("/api/v1/events").send({
      endpointUrl: "https://a.com",
      status: "healthy",
      responseTimeMs: 10,
    });
    await request(app).post("/api/v1/events").send({
      endpointUrl: "https://b.com",
      status: "unhealthy",
      responseTimeMs: 500,
    });

    const res = await request(app).get("/api/v1/events");
    expect(res.body.total).toBe(2);
    expect(res.body.events).toHaveLength(2);
    expect(res.body.events[0].endpointUrl).toBe("https://b.com");
  });

  it("respects limit parameter", async () => {
    for (let i = 0; i < 5; i++) {
      await request(app).post("/api/v1/events").send({
        endpointUrl: `https://site${i}.com`,
        status: "healthy",
        responseTimeMs: 10,
      });
    }
    const res = await request(app).get("/api/v1/events?limit=2");
    expect(res.body.events).toHaveLength(2);
    expect(res.body.total).toBe(5);
  });
});

describe("GET /api/v1/stats", () => {
  it("returns zeros when no events", async () => {
    const res = await request(app).get("/api/v1/stats");
    expect(res.status).toBe(200);
    expect(res.body.totalChecks).toBe(0);
    expect(res.body.uptimePercent).toBe(0);
  });

  it("calculates correct stats", async () => {
    await request(app).post("/api/v1/events").send({
      endpointUrl: "https://a.com",
      status: "healthy",
      responseTimeMs: 100,
    });
    await request(app).post("/api/v1/events").send({
      endpointUrl: "https://a.com",
      status: "healthy",
      responseTimeMs: 200,
    });
    await request(app).post("/api/v1/events").send({
      endpointUrl: "https://a.com",
      status: "unhealthy",
      responseTimeMs: 500,
    });

    const res = await request(app).get("/api/v1/stats");
    expect(res.body.totalChecks).toBe(3);
    expect(res.body.healthyCount).toBe(2);
    expect(res.body.unhealthyCount).toBe(1);
    expect(res.body.avgResponseTimeMs).toBe(267);
    expect(res.body.uptimePercent).toBe(66.67);
  });

  it("filters by url", async () => {
    await request(app).post("/api/v1/events").send({
      endpointUrl: "https://a.com",
      status: "healthy",
      responseTimeMs: 100,
    });
    await request(app).post("/api/v1/events").send({
      endpointUrl: "https://b.com",
      status: "unhealthy",
      responseTimeMs: 500,
    });

    const res = await request(app).get("/api/v1/stats?url=https://a.com");
    expect(res.body.totalChecks).toBe(1);
    expect(res.body.healthyCount).toBe(1);
    expect(res.body.uptimePercent).toBe(100);
  });
});

describe("DELETE /api/v1/events", () => {
  it("clears all events", async () => {
    await request(app).post("/api/v1/events").send({
      endpointUrl: "https://a.com",
      status: "healthy",
      responseTimeMs: 10,
    });

    const res = await request(app).delete("/api/v1/events");
    expect(res.status).toBe(200);
    expect(res.body.count).toBe(1);

    const list = await request(app).get("/api/v1/events");
    expect(list.body.total).toBe(0);
  });
});
