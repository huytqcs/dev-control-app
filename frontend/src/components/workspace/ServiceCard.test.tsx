import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ServiceCard } from "@/components/workspace/ServiceCard";
import type { ServiceDTO } from "@/types/api";

function makeService(overrides: Partial<ServiceDTO> = {}): ServiceDTO {
  return {
    id: "svc-1",
    name: "api-gateway",
    type: "node",
    path: "/repo/api-gateway",
    port: 4312,
    openUrls: [],
    dependsOn: [],
    state: {
      status: "stopped",
      git: { branch: "main", dirty: false, ahead: 0, behind: 0 },
      health: { status: "unknown" },
    },
    actions: [],
    workers: [],
    ...overrides,
  };
}

function renderCard(
  service: ServiceDTO,
  overrides: Partial<{
    isSelected: boolean;
    onSelect: () => void;
    onStart: () => void;
    onStop: () => void;
    onRestart: () => void;
    pending: boolean;
  }> = {},
) {
  const onSelect = overrides.onSelect ?? vi.fn();
  const onStart = overrides.onStart ?? vi.fn();
  const onStop = overrides.onStop ?? vi.fn();
  const onRestart = overrides.onRestart ?? vi.fn();

  render(
    <ServiceCard
      service={service}
      isSelected={overrides.isSelected ?? false}
      onSelect={onSelect}
      onStart={onStart}
      onStop={onStop}
      onRestart={onRestart}
      pending={overrides.pending ?? false}
    />,
  );

  return { onSelect, onStart, onStop, onRestart };
}

describe("ServiceCard", () => {
  it("renders the service name", () => {
    renderCard(makeService({ name: "worker-queue" }));

    expect(screen.getByText("worker-queue")).toBeInTheDocument();
  });

  it("shows a Start button when stopped and calls onStart when clicked", async () => {
    const user = userEvent.setup();
    const { onStart, onStop } = renderCard(
      makeService({ state: { status: "stopped", git: { branch: "main", dirty: false, ahead: 0, behind: 0 }, health: { status: "unknown" } } }),
    );

    const button = screen.getByRole("button", { name: "Start" });
    expect(button).toBeInTheDocument();

    await user.click(button);

    expect(onStart).toHaveBeenCalledTimes(1);
    expect(onStop).not.toHaveBeenCalled();
  });

  it("shows a Stop button when running and calls onStop when clicked", async () => {
    const user = userEvent.setup();
    const { onStart, onStop } = renderCard(
      makeService({ state: { status: "running", git: { branch: "main", dirty: false, ahead: 0, behind: 0 }, health: { status: "unknown" } } }),
    );

    const button = screen.getByRole("button", { name: "Stop" });
    expect(button).toBeInTheDocument();

    await user.click(button);

    expect(onStop).toHaveBeenCalledTimes(1);
    expect(onStart).not.toHaveBeenCalled();
  });

  it("shows the last error text when state.lastError is set", () => {
    renderCard(
      makeService({
        state: {
          status: "failed",
          git: { branch: "main", dirty: false, ahead: 0, behind: 0 },
          health: { status: "unknown" },
          lastError: "exit code 1: connection refused",
        },
      }),
    );

    expect(
      screen.getByText("exit code 1: connection refused"),
    ).toBeInTheDocument();
  });
});
