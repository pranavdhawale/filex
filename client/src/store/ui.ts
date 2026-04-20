import { create } from "zustand";

interface UIState {
  startupComplete: boolean;
}

export const useUIStore = create<UIState>(() => ({
  startupComplete: false,
}));