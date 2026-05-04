import { app } from "./app";

const port = parseInt(process.env.DASHBOARD_PORT || "8003", 10);

app.listen(port, "0.0.0.0", () => {
  console.log(`[analytics-dashboard] Running on port ${port}`);
});
