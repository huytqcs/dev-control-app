import type { LogEntry } from "@/types/api";

/** How close to the bottom (in px) counts as "at the bottom" for auto-follow purposes. */
export const NEAR_BOTTOM_THRESHOLD_PX = 32;

export type SeverityFilter = "all" | "stdout" | "stderr";

const ERROR_PATTERN = /error|exception|fatal|panic/i;
const WARNING_PATTERN = /warn(ing)?|deprecat/i;

export function isErrorLine(line: string): boolean {
  return ERROR_PATTERN.test(line);
}

export function isWarningLine(line: string): boolean {
  return WARNING_PATTERN.test(line);
}

// eslint-disable-next-line no-control-regex
const ANSI_PATTERN = /\x1b\[[0-9;]*[a-zA-Z]/g;

export function stripAnsi(line: string): string {
  return line.replace(ANSI_PATTERN, "");
}

/** Formats a single log entry the same way it's rendered: "<time> <line>". */
export function formatLogLine(entry: Pick<LogEntry, "time" | "line">): string {
  return `${new Date(entry.time).toLocaleTimeString()} ${stripAnsi(entry.line)}`;
}
