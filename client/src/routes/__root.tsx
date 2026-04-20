import { createRootRoute, Outlet } from "@tanstack/react-router";

export const Route = createRootRoute({
  component: () => (
    <div className="min-h-screen bg-[#050505] text-white">
      <Outlet />
    </div>
  ),
});