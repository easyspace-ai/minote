# API 调用规范

本文档定义 API 服务的封装方式，使用 TanStack Query 进行服务端状态管理。

## 技术栈

- **TanStack Query v5**: 服务端状态管理、缓存、自动重试
- **Fetch API**: 原生请求（可配合 fetch 封装库）

## 项目结构

```
src/
├── lib/
│   └── api.ts          # API 基础配置
├── hooks/
│   ├── use-repositories.ts
│   ├── use-tasks.ts
│   └── use-documents.ts
├── types/
│   └── index.ts        # API 类型定义
└── providers/
    └── query-provider.tsx  # QueryClient 配置
```

## QueryClient 配置

```tsx
// providers/query-provider.tsx
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000,    // 5 分钟后数据过期
      gcTime: 10 * 60 * 1000,      // 10 分钟后垃圾回收
      retry: 3,                     // 失败重试 3 次
      refetchOnWindowFocus: false,  // 窗口聚焦时不自动刷新
    },
  },
});

export function QueryProvider({ children }: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      {children}
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  );
}
```

## API 基础封装

```tsx
// lib/api.ts
const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8080/api';

class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public data?: unknown
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

async function fetchApi<T>(
  endpoint: string,
  options?: RequestInit
): Promise<T> {
  const url = `${API_BASE}${endpoint}`;
  const token = localStorage.getItem('token');
  
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token && { Authorization: `Bearer ${token}` }),
      ...options?.headers,
    },
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new ApiError(
      response.status,
      error.message || response.statusText,
      error
    );
  }

  return response.json();
}

export { fetchApi, ApiError };
```

## Query Hooks 定义

### 查询 Hook（useQuery）

```tsx
// hooks/use-repositories.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { fetchApi } from '@/lib/api';
import type { Repository } from '@/types';

const repositoryKeys = {
  all: ['repositories'] as const,
  lists: () => [...repositoryKeys.all, 'list'] as const,
  list: (filters: string) => [...repositoryKeys.lists(), { filters }] as const,
  details: () => [...repositoryKeys.all, 'detail'] as const,
  detail: (id: number) => [...repositoryKeys.details(), id] as const,
};

// 获取仓库列表
export function useRepositories() {
  return useQuery({
    queryKey: repositoryKeys.lists(),
    queryFn: () => fetchApi<Repository[]>('/repositories'),
  });
}

// 获取单个仓库
export function useRepository(id: number) {
  return useQuery({
    queryKey: repositoryKeys.detail(id),
    queryFn: () => fetchApi<Repository>(`/repositories/${id}`),
    enabled: !!id,
  });
}
```

### 变更 Hook（useMutation）

```tsx
// hooks/use-repositories.ts (续)

export function useCreateRepository() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (url: string) =>
      fetchApi<Repository>('/repositories', {
        method: 'POST',
        body: JSON.stringify({ url }),
      }),
    onSuccess: () => {
      // 成功后刷新仓库列表
      queryClient.invalidateQueries({ queryKey: repositoryKeys.lists() });
    },
  });
}

export function useDeleteRepository() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: number) =>
      fetchApi(`/repositories/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: repositoryKeys.lists() });
    },
  });
}

export function useRunTask() {
  return useMutation({
    mutationFn: (id: number) =>
      fetchApi(`/tasks/${id}/run`, { method: 'POST' }),
  });
}
```

## 在组件中使用

```tsx
import { useRepositories, useDeleteRepository } from '@/hooks/use-repositories';
import { Button } from '@/components/ui/button';
import { toast } from 'sonner';

function RepositoryList() {
  const { data: repos, isLoading, error } = useRepositories();
  const deleteMutation = useDeleteRepository();

  const handleDelete = async (id: number) => {
    try {
      await deleteMutation.mutateAsync(id);
      toast.success('Repository deleted');
    } catch {
      toast.error('Failed to delete repository');
    }
  };

  if (isLoading) return <Skeleton />;
  if (error) return <ErrorMessage error={error} />;

  return (
    <ul>
      {repos?.map((repo) => (
        <li key={repo.id}>
          {repo.name}
          <Button
            variant="destructive"
            onClick={() => handleDelete(repo.id)}
            disabled={deleteMutation.isPending}
          >
            {deleteMutation.isPending ? 'Deleting...' : 'Delete'}
          </Button>
        </li>
      ))}
    </ul>
  );
}
```

## 乐观更新

```tsx
// hooks/use-update-repository.ts
export function useUpdateRepository() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<Repository> }) =>
      fetchApi<Repository>(`/repositories/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(data),
      }),
    // 乐观更新
    onMutate: async ({ id, data }) => {
      await queryClient.cancelQueries({ queryKey: repositoryKeys.detail(id) });
      const previousRepo = queryClient.getQueryData<Repository>(
        repositoryKeys.detail(id)
      );
      queryClient.setQueryData(repositoryKeys.detail(id), (old) => ({
        ...old,
        ...data,
      }));
      return { previousRepo };
    },
    onError: (_err, { id }, context) => {
      // 回滚
      queryClient.setQueryData(
        repositoryKeys.detail(id),
        context?.previousRepo
      );
    },
    onSettled: (_, __, { id }) => {
      queryClient.invalidateQueries({ queryKey: repositoryKeys.detail(id) });
    },
  });
}
```

## 分页查询

```tsx
// hooks/use-documents.ts
import { useInfiniteQuery } from '@tanstack/react-query';

export function useDocuments(repoId: number) {
  return useInfiniteQuery({
    queryKey: ['documents', repoId],
    queryFn: ({ pageParam = 1 }) =>
      fetchApi<PaginatedResponse<Document>>(
        `/repositories/${repoId}/documents?page=${pageParam}`
      ),
    getNextPageParam: (lastPage) =>
      lastPage.page < lastPage.total_pages ? lastPage.page + 1 : undefined,
    initialPageParam: 1,
  });
}

// 组件中使用
function DocumentList({ repoId }: { repoId: number }) {
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useDocuments(repoId);

  return (
    <>
      {data?.pages.map((page) =>
        page.data.map((doc) => <DocumentCard key={doc.id} doc={doc} />)
      )}
      {hasNextPage && (
        <Button
          onClick={() => fetchNextPage()}
          disabled={isFetchingNextPage}
        >
          {isFetchingNextPage ? 'Loading...' : 'Load More'}
        </Button>
      )}
    </>
  );
}
```

## API 命名规范

| 操作       | HTTP 方法 | 命名   | 示例                             |
| ---------- | --------- | ------ | -------------------------------- |
| 获取列表   | GET       | list   | `repositoryApi.list()`           |
| 获取单个   | GET       | get    | `repositoryApi.get(id)`          |
| 创建       | POST      | create | `repositoryApi.create(data)`     |
| 更新       | PUT/PATCH | update | `repositoryApi.update(id, data)` |
| 删除       | DELETE    | delete | `repositoryApi.delete(id)`       |
| 自定义操作 | POST      | 动词   | `taskApi.run(id)`                |

## 相关文档

- [类型定义规范](./10-类型定义规范.md)
- [状态管理规范](./06-状态管理规范.md)
