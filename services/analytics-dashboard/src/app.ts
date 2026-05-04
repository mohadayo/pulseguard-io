import express, { Request, Response } from "express";
import cors from "cors";

const app = express();
app.use(cors());
app.use(express.json());

const startTime = Date.now();

interface CheckEvent {
  id: string;
  endpointUrl: string;
  status: string;
  statusCode?: number;
  responseTimeMs: number;
  checkedAt: string;
}

interface StatsResult {
  totalChecks: number;
  healthyCount: number;
  unhealthyCount: number;
  avgResponseTimeMs: number;
  uptimePercent: number;
}

const eventsStore: CheckEvent[] = [];

let eventCounter = 0;

app.get("/health", (_req: Request, res: Response) => {
  res.json({
    status: "healthy",
    service: "analytics-dashboard",
    uptime_seconds: Math.round((Date.now() - startTime) / 1000),
    timestamp: new Date().toISOString(),
  });
});

app.post("/api/v1/events", (req: Request, res: Response) => {
  const { endpointUrl, status, statusCode, responseTimeMs } = req.body;

  if (!endpointUrl || !status) {
    res.status(400).json({ error: "endpointUrl and status are required" });
    return;
  }

  if (typeof responseTimeMs !== "number" || responseTimeMs < 0) {
    res.status(400).json({ error: "responseTimeMs must be a non-negative number" });
    return;
  }

  eventCounter++;
  const event: CheckEvent = {
    id: `evt-${eventCounter}`,
    endpointUrl,
    status,
    statusCode,
    responseTimeMs,
    checkedAt: new Date().toISOString(),
  };

  eventsStore.push(event);
  console.log(`[analytics] Recorded event ${event.id} for ${endpointUrl}: ${status}`);
  res.status(201).json(event);
});

app.get("/api/v1/events", (_req: Request, res: Response) => {
  const limit = Math.min(parseInt(_req.query.limit as string) || 50, 200);
  const recent = eventsStore.slice(-limit).reverse();
  res.json({ events: recent, total: eventsStore.length });
});

app.get("/api/v1/stats", (_req: Request, res: Response) => {
  const url = _req.query.url as string | undefined;

  let filtered = eventsStore;
  if (url) {
    filtered = eventsStore.filter((e) => e.endpointUrl === url);
  }

  if (filtered.length === 0) {
    res.json({
      totalChecks: 0,
      healthyCount: 0,
      unhealthyCount: 0,
      avgResponseTimeMs: 0,
      uptimePercent: 0,
    } as StatsResult);
    return;
  }

  const healthyCount = filtered.filter((e) => e.status === "healthy").length;
  const unhealthyCount = filtered.length - healthyCount;
  const avgResponseTimeMs =
    Math.round(
      filtered.reduce((sum, e) => sum + e.responseTimeMs, 0) / filtered.length
    );
  const uptimePercent =
    Math.round((healthyCount / filtered.length) * 10000) / 100;

  const stats: StatsResult = {
    totalChecks: filtered.length,
    healthyCount,
    unhealthyCount,
    avgResponseTimeMs,
    uptimePercent,
  };

  res.json(stats);
});

app.delete("/api/v1/events", (_req: Request, res: Response) => {
  const count = eventsStore.length;
  eventsStore.length = 0;
  eventCounter = 0;
  console.log(`[analytics] Cleared ${count} events`);
  res.json({ message: "cleared", count });
});

export { app, eventsStore, CheckEvent };
