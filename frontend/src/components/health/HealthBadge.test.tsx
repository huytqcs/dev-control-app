import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { HealthBadge } from "@/components/health/HealthBadge";

describe("HealthBadge", () => {
  it("renders nothing for unknown status", () => {
    render(<HealthBadge status="unknown" serviceStatus="running" />);
    expect(screen.queryByText(/healthy/i)).not.toBeInTheDocument();
  });

  it("shows the healthy label for a running service", () => {
    render(<HealthBadge status="healthy" serviceStatus="running" />);
    expect(screen.getByText("Healthy")).toBeInTheDocument();
  });

  it("still shows the badge for a running service with a real unhealthy check", () => {
    render(<HealthBadge status="unhealthy" serviceStatus="running" />);
    expect(screen.getByText("Unhealthy")).toBeInTheDocument();
  });

  // Each of these pairs a real (non-"running") serviceStatus with a stale
  // healthy/unhealthy value that the backend hasn't gotten around to
  // resetting yet (or a dropped WS event never told the frontend about) —
  // the badge must hide regardless, since health only ever means anything
  // while genuinely running.
  it.each(["stopping", "stopped", "starting", "failed"] as const)(
    "hides for serviceStatus=%s even with a stale healthy/unhealthy status",
    (serviceStatus) => {
      render(<HealthBadge status="unhealthy" serviceStatus={serviceStatus} />);
      expect(screen.queryByText(/healthy/i)).not.toBeInTheDocument();
    },
  );
});
