// Shared theme hook — single source of truth used by every component that
// needs to read or change the active light/dark palette. All instances stay
// in sync via a custom DOM event, and the hook also reacts to OS-level
// prefers-color-scheme changes when the user has no saved preference.

import { useCallback, useEffect, useState } from "react";
import {
  getCurrentTheme,
  setTheme as persistTheme,
  THEME_STORAGE_KEY,
  type Theme,
} from "@/lib/theme";

const THEME_CHANGE_EVENT = "gitmap:theme-change";

interface UseThemeResult {
  theme: Theme;
  isDark: boolean;
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;
}

export function useTheme(): UseThemeResult {
  const [theme, setThemeState] = useState<Theme>(() => getCurrentTheme());

  // Sync across hook instances + cross-tab + OS-level changes.
  useEffect(() => {
    const handleThemeEvent = (event: Event) => {
      const next = (event as CustomEvent<Theme>).detail;
      if (next === "light" || next === "dark") setThemeState(next);
    };

    const handleStorage = (event: StorageEvent) => {
      if (event.key !== THEME_STORAGE_KEY) return;
      if (event.newValue === "light" || event.newValue === "dark") {
        setThemeState(event.newValue);
        persistTheme(event.newValue);
      }
    };

    const mediaQuery = window.matchMedia("(prefers-color-scheme: light)");
    const handleSystemChange = (event: MediaQueryListEvent) => {
      // Only follow the OS when the user hasn't explicitly chosen a theme.
      try {
        if (localStorage.getItem(THEME_STORAGE_KEY)) return;
      } catch {
        return;
      }
      const next: Theme = event.matches ? "light" : "dark";
      setThemeState(next);
      document.documentElement.classList.toggle("dark", next === "dark");
      document.documentElement.classList.toggle("light", next === "light");
    };

    window.addEventListener(THEME_CHANGE_EVENT, handleThemeEvent);
    window.addEventListener("storage", handleStorage);
    mediaQuery.addEventListener("change", handleSystemChange);

    return () => {
      window.removeEventListener(THEME_CHANGE_EVENT, handleThemeEvent);
      window.removeEventListener("storage", handleStorage);
      mediaQuery.removeEventListener("change", handleSystemChange);
    };
  }, []);

  const setTheme = useCallback((next: Theme) => {
    persistTheme(next);
    setThemeState(next);
    window.dispatchEvent(new CustomEvent<Theme>(THEME_CHANGE_EVENT, { detail: next }));
  }, []);

  const toggleTheme = useCallback(() => {
    setTheme(theme === "dark" ? "light" : "dark");
  }, [theme, setTheme]);

  return { theme, isDark: theme === "dark", setTheme, toggleTheme };
}
