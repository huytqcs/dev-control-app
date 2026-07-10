import type { LogEntry, ServiceDTO, WorkspaceDTO } from "@/types/api";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, init);
  if (!res.ok) {
    let message = `${res.status} ${res.statusText}`;
    try {
      const body = await res.json();
      if (body?.error?.message) message = body.error.message;
    } catch {
      // non-JSON error body, keep the status text
    }
    throw new Error(message);
  }
  return res.json() as Promise<T>;
}

export function getWorkspace(): Promise<WorkspaceDTO> {
  return request<WorkspaceDTO>("/api/workspace");
}

export async function getServices(): Promise<ServiceDTO[]> {
  const data = await request<{ services: ServiceDTO[] }>("/api/services");
  return data.services;
}

export function getService(id: string): Promise<ServiceDTO> {
  return request<ServiceDTO>(`/api/services/${id}`);
}

export function startService(id: string): Promise<ServiceDTO> {
  return request<ServiceDTO>(`/api/services/${id}/start`, { method: "POST" });
}

export function stopService(id: string): Promise<ServiceDTO> {
  return request<ServiceDTO>(`/api/services/${id}/stop`, { method: "POST" });
}

export function restartService(id: string): Promise<ServiceDTO> {
  return request<ServiceDTO>(`/api/services/${id}/restart`, {
    method: "POST",
  });
}

export async function getServiceLogs(
  id: string,
  limit = 500,
): Promise<LogEntry[]> {
  const data = await request<{ entries: LogEntry[] }>(
    `/api/services/${id}/logs?limit=${limit}`,
  );
  return data.entries;
}
