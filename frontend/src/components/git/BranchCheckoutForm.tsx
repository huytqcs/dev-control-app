import { useState } from "react";
import { Button } from "@/components/ui/button";

export function BranchCheckoutForm({
  onCheckout,
  pending,
}: {
  onCheckout: (branch: string) => void;
  pending: boolean;
}) {
  const [branch, setBranch] = useState("");

  return (
    <form
      className="flex gap-2"
      onSubmit={(e) => {
        e.preventDefault();
        if (!branch.trim()) return;
        onCheckout(branch.trim());
      }}
    >
      <input
        value={branch}
        onChange={(e) => setBranch(e.target.value)}
        placeholder="branch name"
        disabled={pending}
        className="h-8 min-w-0 flex-1 rounded-md border bg-background px-2 text-sm outline-none focus-visible:ring-1 focus-visible:ring-ring"
      />
      <Button type="submit" size="sm" variant="outline" disabled={pending || !branch.trim()}>
        Checkout
      </Button>
    </form>
  );
}
