import { lazy } from "react";
import { createBrowserRouter } from "react-router-dom";

import { AppShell } from "@/layouts/AppShell";
import Forbidden from "@/pages/Forbidden";
import Login from "@/pages/Login";
import NotFound from "@/pages/NotFound";

import { RequireAdmin, RequireAuth } from "./guards";

// 重页面懒加载,拆分 chunk;登录/错误页保持同步以秒开
const Dashboard = lazy(() => import("@/pages/Dashboard"));
const Zones = lazy(() => import("@/pages/Zones"));
const Domains = lazy(() => import("@/pages/Domains"));
const Records = lazy(() => import("@/pages/Records"));
const Users = lazy(() => import("@/pages/Users"));
const Me = lazy(() => import("@/pages/Me"));

export const router = createBrowserRouter([
  { path: "/login", element: <Login /> },
  { path: "/403", element: <Forbidden /> },
  {
    element: (
      <RequireAuth>
        <AppShell />
      </RequireAuth>
    ),
    children: [
      { index: true, element: <Dashboard /> },
      { path: "dns/zones", element: <Zones /> },
      { path: "dns/zones/:zone/domains", element: <Domains /> },
      {
        path: "dns/zones/:zone/domains/:domain/records",
        element: <Records />,
      },
      {
        path: "admin/users",
        element: (
          <RequireAdmin>
            <Users />
          </RequireAdmin>
        ),
      },
      { path: "me", element: <Me /> },
    ],
  },
  { path: "*", element: <NotFound /> },
]);
