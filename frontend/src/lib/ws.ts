import type { AppEvent } from "@/types/api";

type Listener = (event: AppEvent) => void;

const RECONNECT_DELAY_MS = 1500;

// Single, app-lifetime WebSocket connection shared across the app
// (ARCHITECTURE.md §6.3 — event-driven, not polling). connect() is
// idempotent and the socket is never explicitly torn down by callers: this
// is a page-lifetime singleton, not something to open/close per component.
// (An earlier refcounted connect/disconnect design broke under React
// StrictMode's dev-mode double-invoke of effects — two independent hooks
// decrementing/closing a shared refcount could each legitimately think they
// were "the last consumer" and close a socket another hook still needed,
// leaving a stale-but-still-delivering socket alongside the new one, which
// double-processed every broadcast.)
class RealtimeClient {
  private socket: WebSocket | null = null;
  private connecting = false;
  private listeners = new Set<Listener>();

  connect() {
    if (this.socket || this.connecting) return;
    this.connecting = true;
    this.open();
  }

  private open() {
    const proto = window.location.protocol === "https:" ? "wss" : "ws";
    const socket = new WebSocket(`${proto}://${window.location.host}/ws`);
    this.socket = socket;

    socket.addEventListener("open", () => {
      this.connecting = false;
    });

    socket.addEventListener("message", (ev) => {
      try {
        const parsed = JSON.parse(ev.data) as AppEvent;
        this.listeners.forEach((listener) => listener(parsed));
      } catch {
        // ignore malformed frames
      }
    });

    socket.addEventListener("close", () => {
      this.socket = null;
      this.connecting = false;
      setTimeout(() => this.open(), RECONNECT_DELAY_MS);
    });
  }

  subscribe(listener: Listener): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }
}

export const realtimeClient = new RealtimeClient();
