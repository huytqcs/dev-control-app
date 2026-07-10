import type { LogEntry } from "@/types/api";

/** How close to the bottom (in px) counts as "at the bottom" for auto-follow purposes. */
export const NEAR_BOTTOM_THRESHOLD_PX = 32;

export type SeverityFilter = "all" | "stdout" | "stderr";

// Matches common error indicators in log text regardless of which stream they
// came from — plenty of real error output goes to stdout depending on the
// app's own logger, so this is independent of entry.source.
const ERROR_PATTERN = /error|exception|fatal|panic/i;

export function isErrorLine(line: string): boolean {
  return ERROR_PATTERN.test(line);
}

/** Formats a single log entry the same way it's rendered: "<time> <line>". */
export function formatLogLine(entry: Pick<LogEntry, "time" | "line">): string {
  return `${new Date(entry.time).toLocaleTimeString()} ${entry.line}`;
}
