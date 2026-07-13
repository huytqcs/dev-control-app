import { useEffect, useState } from "react";
import { realtimeClient, type ConnectionStatus } from "@/lib/ws";

export function useConnectionStatus(): ConnectionStatus {
  const [status, setStatus] = useState<ConnectionStatus>(realtimeClient.getStatus());

  useEffect(() => realtimeClient.onStatusChange(setStatus), []);

  return status;
}
