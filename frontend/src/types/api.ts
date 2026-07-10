export type ServiceStatus =
  | "stopped"
  | "starting"
  | "running"
  | "failed"
  | "stopping";

export interface GitStateDTO {
  branch: string;
  dirty: boolean;
  ahead: number;
  behind: number;
}

export type HealthStatus = "unknown" | "healthy" | "unhealthy";

export interface HealthStateDTO {
  status: HealthStatus;
}

export interface ServiceStateDTO {
  status: ServiceStatus;
  pid?: number;
  startedAt?: string;
  lastError?: string;
  lastExitCode?: number;
  git: GitStateDTO;
  health: HealthStateDTO;
}

export interface ActionSummaryDTO {
  id: string;
  name: string;
}

export interface WorkerSummaryDTO {
  id: string;
  name: string;
  status: string;
  pid?: number;
  lastError?: string;
  lastExitCode?: number;
}

export interface ServiceDTO {
  id: string;
  name: string;
  type: string;
  path: string;
  port: number;
  openUrls: string[];
  dependsOn: string[];
  state: ServiceStateDTO;
  actions: ActionSummaryDTO[];
  workers: WorkerSummaryDTO[];
}

export interface PresetDTO {
  id: string;
  name: string;
  services: string[];
}

export interface WorkspaceDTO {
  name: string;
  presets: PresetDTO[];
}

export interface LogEntry {
  id: string;
  streamKey: string;
  source: "stdout" | "stderr";
  line: string;
  time: string;
  level?: string;
}

// The WebSocket service.updated payload is the backend's raw runtime.ServiceState,
// which is a different (narrower) shape than ServiceDTO — it has no
// type/path/openUrls/dependsOn/actions, since those come from config, not
// runtime state. See internal/runtime/process_state.go on the backend.
export interface RuntimeWorkerState {
  id: string;
  name: string;
  status: string;
  pid?: number;
  lastError?: string;
  lastExitCode?: number;
}

export interface RuntimeServiceState {
  id: string;
  name: string;
  status: ServiceStatus;
  pid?: number;
  port?: number;
  startedAt?: string;
  lastError?: string;
  lastExitCode?: number;
  git: GitStateDTO;
  health: HealthStateDTO;
  workers: RuntimeWorkerState[];
}

export interface ServiceUpdatedEvent {
  type: "service.updated";
  serviceId: string;
  payload: RuntimeServiceState;
  time: string;
}

export interface LogAppendedEvent {
  type: "log.appended";
  serviceId: string;
  payload: { entry: LogEntry };
  time: string;
}

export interface HealthUpdatedEvent {
  type: "health.updated";
  serviceId: string;
  payload: { health: HealthStateDTO };
  time: string;
}

export interface GitUpdatedEvent {
  type: "git.updated";
  serviceId: string;
  payload: { git: GitStateDTO };
  time: string;
}

export interface WorkerUpdatedEvent {
  type: "worker.updated";
  serviceId: string;
  payload: { worker: WorkerSummaryDTO };
  time: string;
}

export type AppEvent =
  | ServiceUpdatedEvent
  | LogAppendedEvent
  | HealthUpdatedEvent
  | GitUpdatedEvent
  | WorkerUpdatedEvent;
