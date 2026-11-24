export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly fields: { field: string; reason: string }[];

  constructor(status: number, code: string, message: string, fields: { field: string; reason: string }[] = []) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.fields = fields;
  }

  get isAuth(): boolean {
    return this.status === 401 || this.code.startsWith('auth.');
  }

  get isNetwork(): boolean {
    return this.status === 0;
  }

  get isPermission(): boolean {
    return this.code === 'permission.denied';
  }
}

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE';
  body?: unknown;
  query?: Record<string, string | number | boolean | null | undefined>;
  signal?: AbortSignal;
  idempotencyKey?: string;
  rawBody?: BodyInit;
  headers?: Record<string, string>;
}

const BASE = '/api/v1';

function buildURL(path: string, query?: RequestOptions['query']): string {
  const url = path.startsWith('/') ? `${BASE}${path}` : `${BASE}/${path}`;
  if (!query) return url;
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined && value !== null && value !== '') params.set(key, String(value));
  }
  const s = params.toString();
  return s ? `${url}?${s}` : url;
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json', ...options.headers };
  let body = options.rawBody;
  if (options.body !== undefined) {
    headers['Content-Type'] = 'application/json';
    body = JSON.stringify(options.body);
  }
  if (options.idempotencyKey) headers['Idempotency-Key'] = options.idempotencyKey;
  let res: Response;
  try {
    res = await fetch(buildURL(path, options.query), {
      method: options.method ?? 'GET',
      headers,
      body,
      credentials: 'same-origin',
      signal: options.signal,
    });
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err;
    throw new ApiError(0, 'network.failed', '网络连接失败，请检查网络后重试');
  }
  if (res.status === 204) return undefined as T;
  const contentType = res.headers.get('Content-Type') ?? '';
  const payload = contentType.includes('application/json') ? await res.json() : await res.text();
  if (!res.ok) {
    const detail = typeof payload === 'object' && payload !== null && 'error' in payload
      ? (payload as { error: { code: string; message: string; fields?: { field: string; reason: string }[] } }).error
      : { code: 'server.unavailable', message: typeof payload === 'string' ? payload : '服务暂时不可用' };
    if (res.status === 401 && typeof window !== 'undefined') {
      window.dispatchEvent(new CustomEvent('mymoments:unauthorized'));
    }
    throw new ApiError(res.status, detail.code, detail.message, detail.fields);
  }
  return payload as T;
}

export const api = {
  get<T>(path: string, query?: RequestOptions['query'], signal?: AbortSignal): Promise<T> {
    return request<T>(path, { query, signal });
  },
  post<T>(path: string, body?: unknown, idempotencyKey?: string): Promise<T> {
    return request<T>(path, { method: 'POST', body, idempotencyKey: idempotencyKey ?? newKey() });
  },
  patch<T>(path: string, body?: unknown): Promise<T> {
    return request<T>(path, { method: 'PATCH', body });
  },
  put<T>(path: string, body?: unknown): Promise<T> {
    return request<T>(path, { method: 'PUT', body });
  },
  del<T>(path: string): Promise<T> {
    return request<T>(path, { method: 'DELETE' });
  },
  upload<T>(path: string, form: FormData, signal?: AbortSignal): Promise<T> {
    return request<T>(path, { method: 'POST', rawBody: form, signal });
  },
};

export function newKey(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) return crypto.randomUUID();
  return `k-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}
