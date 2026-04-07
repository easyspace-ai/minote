import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Loader2, Puzzle, RefreshCw, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { chatclawApi } from "@/api/chatclaw";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";

/**
 * 侧栏「技能」：扫描仓库/数据目录下的 SKILL.md，经 /api/v1/skills/* 持久化启用状态。
 */
export function SkillsPage() {
  const qc = useQueryClient();

  const installed = useQuery({
    queryKey: ["skills-installed"],
    queryFn: () => chatclawApi.skills.installed(),
  });

  const workspace = useQuery({
    queryKey: ["skills-workspace"],
    queryFn: () => chatclawApi.skills.workspace(),
  });

  const refresh = useMutation({
    mutationFn: () => chatclawApi.skills.refresh(),
    onSuccess: (list) => {
      qc.setQueryData(["skills-installed"], list);
      toast.success("已刷新技能列表");
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "刷新失败"),
  });

  const setEnabled = useMutation({
    mutationFn: async ({ slug, enabled }: { slug: string; enabled: boolean }) => {
      if (enabled) {
        await chatclawApi.skills.enable(slug);
      } else {
        await chatclawApi.skills.disable(slug);
      }
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["skills-installed"] });
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "更新失败"),
  });

  const uninstall = useMutation({
    mutationFn: (slug: string) => chatclawApi.skills.uninstall(slug),
    onSuccess: () => {
      toast.success("已移除技能");
      void qc.invalidateQueries({ queryKey: ["skills-installed"] });
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "移除失败"),
  });

  const list = installed.data ?? [];
  const busy = installed.isLoading || refresh.isPending;

  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden bg-white">
      <div className="flex h-14 shrink-0 items-center justify-between gap-3 border-b border-black/[0.06] px-6">
        <h1 className="text-foreground flex items-center gap-2 text-lg font-bold tracking-tight">
          <Puzzle className="h-5 w-5" />
          技能
        </h1>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="gap-1.5"
          disabled={busy}
          onClick={() => refresh.mutate()}
        >
          {refresh.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
          刷新
        </Button>
      </div>
      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto p-6">
        {workspace.data?.skills_dir ? (
          <p className="text-muted-foreground max-w-3xl text-xs leading-relaxed">
            扫描目录（可通过环境变量 <code className="text-foreground/90">NOTEX_SKILLS_PATH</code> 配置，逗号分隔多个路径）：
            <br />
            <code className="text-foreground/80 mt-1 inline-block break-all">{workspace.data.skills_dir}</code>
          </p>
        ) : null}

        {installed.isLoading && (
          <p className="text-muted-foreground flex items-center gap-2 text-sm">
            <Loader2 className="h-4 w-4 animate-spin" /> 加载中…
          </p>
        )}
        {installed.error && (
          <p className="text-destructive text-sm">
            无法加载技能列表，请确认后端已启动并已登录。
          </p>
        )}
        {!installed.isLoading && !installed.error && list.length === 0 && (
          <p className="text-muted-foreground max-w-md text-sm leading-relaxed">
            未发现 SKILL.md。请将技能放在仓库根目录的 <code className="text-foreground/80">skills/</code> 下（或{" "}
            <code className="text-foreground/80">$NOTEX_DATA_ROOT/skills</code>
            ），然后点击「刷新」。
          </p>
        )}

        <ul className="max-w-3xl space-y-2">
          {list.map((s) => {
            const isCustom = s.source === "custom";
            return (
              <li
                key={s.slug}
                className="flex flex-col gap-2 rounded-2xl border border-black/[0.06] bg-white px-4 py-3 shadow-sm sm:flex-row sm:items-start sm:justify-between"
              >
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="text-foreground font-semibold">{s.name}</span>
                    <span className="text-muted-foreground rounded bg-black/[0.04] px-1.5 py-px font-mono text-[10px]">
                      {s.source}
                    </span>
                    {s.version ? (
                      <span className="text-muted-foreground text-[10px]">v{s.version}</span>
                    ) : null}
                  </div>
                  <span className="text-muted-foreground font-mono text-xs">{s.slug}</span>
                  {s.description ? (
                    <p className="text-muted-foreground mt-1 text-sm leading-relaxed">{s.description}</p>
                  ) : null}
                  <p className="text-muted-foreground/80 mt-2 text-[10px]">
                    安装时间 {s.installedAt ? new Date(s.installedAt).toLocaleString() : "—"}
                  </p>
                </div>
                <div className="flex shrink-0 items-center gap-3 sm:flex-col sm:items-end">
                  <div className="flex items-center gap-2">
                    <span className="text-muted-foreground text-xs">启用</span>
                    <Switch
                      checked={s.enabled}
                      disabled={setEnabled.isPending}
                      onCheckedChange={(checked) =>
                        setEnabled.mutate({ slug: s.slug, enabled: checked })
                      }
                    />
                  </div>
                  {isCustom ? (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive h-8 gap-1 px-2"
                      disabled={uninstall.isPending}
                      onClick={() => {
                        if (window.confirm(`确定移除自定义技能「${s.name}」？将从磁盘删除该目录。`)) {
                          uninstall.mutate(s.slug);
                        }
                      }}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                      移除
                    </Button>
                  ) : null}
                </div>
              </li>
            );
          })}
        </ul>
      </div>
    </div>
  );
}
