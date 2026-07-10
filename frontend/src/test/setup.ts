// Runs before every test file (wired in via `test.setupFiles` in vite.config.ts).
// The `/vitest` entry point both registers the jest-dom matchers on
// vitest's `expect` at runtime and augments vitest's `Assertion` type so
// `.toBeInTheDocument()` etc. type-check.
import "@testing-library/jest-dom/vitest";
import { afterEach } from "vitest";
import { cleanup } from "@testing-library/react";

// `globals` isn't enabled in vite.config.ts (keeps tsconfig untouched), so
// React Testing Library's automatic cleanup-on-afterEach never registers
// itself. Do it explicitly so each test starts from an empty DOM.
afterEach(() => {
  cleanup();
});
