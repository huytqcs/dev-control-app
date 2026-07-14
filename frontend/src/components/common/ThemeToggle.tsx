import { Monitor, Moon, Sun } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useTheme, type ThemeMode } from "@/hooks/useTheme";

const NEXT: Record<ThemeMode, ThemeMode> = {
  system: "light",
  light: "dark",
  dark: "system",
};

const ICON: Record<ThemeMode, typeof Sun> = {
  system: Monitor,
  light: Sun,
  dark: Moon,
};

const LABEL: Record<ThemeMode, string> = {
  system: "Theme: system",
  light: "Theme: light",
  dark: "Theme: dark",
};

export function ThemeToggle() {
  const { mode, setMode } = useTheme();
  const Icon = ICON[mode];

  return (
    <Button
      type="button"
      size="icon-sm"
      variant="ghost"
      title={`${LABEL[mode]} (click to change)`}
      onClick={() => setMode(NEXT[mode])}
    >
      <Icon />
    </Button>
  );
}
