/** `AppShell` 通过 `<Outlet context={...} />` 传给首页等子路由 */
export type AppShellOutletContext = {
  openNewProject: () => void;
  /** 递增后 `WorkspacePage` 应聚焦搜索框 */
  searchFocusNonce: number;
};
