import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { LogViewer } from "@/components/logs/LogViewer";
import { getServiceLogs } from "@/lib/api";
import { realtimeClient } from "@/lib/ws";
import type { LogEntry } from "@/types/api";

// LogViewer renders through the real useServiceLogs hook, which talks to
// getServiceLogs() (fetch) and realtimeClient (a real WebSocket) — mock both
// so the test does no network/socket activity.
vi.mock("@/lib/api", () => ({
  getServiceLogs: vi.fn(),
}));

vi.mock("@/lib/ws", () => ({
  realtimeClient: {
    connect: vi.fn(),
    subscribe: vi.fn(() => () => {}),
  },
}));

const mockedGetServiceLogs = vi.mocked(getServiceLogs);
const mockedConnect = vi.mocked(realtimeClient.connect);
const mockedSubscribe = vi.mocked(realtimeClient.subscribe);

function makeEntry(overrides: Partial<LogEntry> = {}): LogEntry {
  return {
    id: "log-1",
    streamKey: "svc-1:stdout",
    source: "stdout",
    line: "server listening on :4312",
    time: "2026-07-10T00:00:00.000Z",
    ...overrides,
  };
}

describe("LogViewer", () => {
  beforeEach(() => {
    mockedConnect.mockReset();
    mockedSubscribe.mockReset();
    mockedSubscribe.mockReturnValue(() => {});
    mockedGetServiceLogs.mockReset();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows a loading state before logs resolve", () => {
    // Never resolve during this test so the loading branch stays visible.
    mockedGetServiceLogs.mockReturnValue(new Promise(() => {}));

    render(<LogViewer serviceId="svc-1" />);

    expect(screen.getByText("Loading logs…")).toBeInTheDocument();
  });

  it("shows an empty state when there are no logs", async () => {
    mockedGetServiceLogs.mockResolvedValue([]);

    render(<LogViewer serviceId="svc-1" />);

    expect(await screen.findByText("No logs yet")).toBeInTheDocument();
    expect(
      screen.getByText("Start the service to see output here."),
    ).toBeInTheDocument();
  });

  it("renders log lines, distinguishing stderr from stdout", async () => {
    mockedGetServiceLogs.mockResolvedValue([
      makeEntry({ id: "1", source: "stdout", line: "booted" }),
      makeEntry({ id: "2", source: "stderr", line: "boom" }),
    ]);

    render(<LogViewer serviceId="svc-1" />);

    // Each log line's text is a direct text-node child of its own row div
    // (alongside a <span> for the timestamp), so `findByText` resolves to
    // that row div itself — see LogViewer.tsx's `cn(...)` on the row.
    const stdoutLine = await screen.findByText("booted");
    const stderrLine = await screen.findByText("boom");

    // stdout line: no red-text class applied
    expect(stdoutLine).not.toHaveClass("text-red-500");
    // stderr line: distinguished with the destructive red-text class
    expect(stderrLine).toHaveClass("text-red-500");
  });

  it("connects and subscribes to the realtime client", async () => {
    mockedGetServiceLogs.mockResolvedValue([]);

    render(<LogViewer serviceId="svc-1" />);

    await waitFor(() => expect(mockedConnect).toHaveBeenCalled());
    expect(mockedSubscribe).toHaveBeenCalled();
  });
});
