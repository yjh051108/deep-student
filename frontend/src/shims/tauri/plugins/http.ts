// ============================================================
// Tauri → Wails 适配层：@tauri-apps/plugin-http
// ------------------------------------------------------------
// 原版 http 插件提供后端代理式 fetch（绕过 CORS）。
// Wails WebView 下直接用 window.fetch（前端 CORS 由目标服务决定，
// 与后端无关）；保留原版调用形状。
// ============================================================

export type ResponseType = 'text' | 'json' | 'binary' | 'arraybuffer';

export interface HttpOptions {
  method?: string;
  headers?: Record<string, string>;
  body?: unknown;
  responseType?: ResponseType;
  timeout?: number;
  connectTimeout?: number;
  maxRedirections?: number;
}

export interface HttpResponse<T = unknown> {
  url: string;
  status: number;
  ok: boolean;
  headers: Record<string, string>;
  data: T;
  rawHeaders?: Record<string, string[]>;
}

export interface ClientOptions {
  connectTimeout?: number;
  maxRedirections?: number;
  timeout?: number;
  headers?: Record<string, string>;
}

export class Client {
  constructor(private baseUrl?: string, private options?: ClientOptions) {}

  async request<T = unknown>(url: string, options: HttpOptions = {}): Promise<HttpResponse<T>> {
    return request<T>(url, { ...options, baseUrl: this.baseUrl });
  }

  async get<T = unknown>(url: string, options: Omit<HttpOptions, 'method'> = {}): Promise<HttpResponse<T>> {
    return this.request<T>(url, { ...options, method: 'GET' });
  }

  async post<T = unknown>(url: string, body?: unknown, options: Omit<HttpOptions, 'method' | 'body'> = {}): Promise<HttpResponse<T>> {
    return this.request<T>(url, { ...options, method: 'POST', body });
  }

  async put<T = unknown>(url: string, body?: unknown, options: Omit<HttpOptions, 'method' | 'body'> = {}): Promise<HttpResponse<T>> {
    return this.request<T>(url, { ...options, method: 'PUT', body });
  }

  async delete<T = unknown>(url: string, options: Omit<HttpOptions, 'method'> = {}): Promise<HttpResponse<T>> {
    return this.request<T>(url, { ...options, method: 'DELETE' });
  }

  async patch<T = unknown>(url: string, body?: unknown, options: Omit<HttpOptions, 'method' | 'body'> = {}): Promise<HttpResponse<T>> {
    return this.request<T>(url, { ...options, method: 'PATCH', body });
  }

  async head<T = unknown>(url: string, options: Omit<HttpOptions, 'method'> = {}): Promise<HttpResponse<T>> {
    return this.request<T>(url, { ...options, method: 'HEAD' });
  }
}

export function getClient(options: ClientOptions = {}): Client {
  return new Client(undefined, options);
}

export async function fetch<T = unknown>(url: string, options: HttpOptions = {}): Promise<HttpResponse<T>> {
  return request<T>(url, options);
}

async function request<T>(url: string, options: HttpOptions & { baseUrl?: string } = {}): Promise<HttpResponse<T>> {
  const fullUrl = options.baseUrl ? new URL(url, options.baseUrl).toString() : url;
  const method = options.method ?? 'GET';
  const headers = new Headers(options.headers as Record<string, string> | undefined);
  let body: BodyInit | null = null;

  if (options.body !== undefined && options.body !== null) {
    if (typeof options.body === 'string') {
      body = options.body;
    } else if (options.body instanceof ArrayBuffer || ArrayBuffer.isView(options.body)) {
      body = options.body as BodyInit;
    } else if (typeof FormData !== 'undefined' && options.body instanceof FormData) {
      body = options.body;
    } else {
      // JSON 对象
      if (!headers.has('content-type')) {
        headers.set('content-type', 'application/json');
      }
      body = JSON.stringify(options.body);
    }
  }

  const controller = typeof AbortController !== 'undefined' ? new AbortController() : null;
  const timeout = options.timeout ?? options.connectTimeout;
  let timer: ReturnType<typeof setTimeout> | undefined;
  if (controller && timeout && timeout > 0) {
    timer = setTimeout(() => controller.abort(), timeout);
  }

  try {
    const resp = await window.fetch(fullUrl, {
      method,
      headers,
      body,
      signal: controller?.signal,
    });
    const respHeaders: Record<string, string> = {};
    resp.headers.forEach((v, k) => {
      respHeaders[k] = v;
    });
    const responseType = options.responseType ?? 'text';
    let data: unknown;
    if (responseType === 'json') {
      data = await resp.json().catch(() => null);
    } else if (responseType === 'arraybuffer' || responseType === 'binary') {
      data = await resp.arrayBuffer();
    } else {
      data = await resp.text();
    }
    return {
      url: resp.url,
      status: resp.status,
      ok: resp.ok,
      headers: respHeaders,
      data: data as T,
    };
  } finally {
    if (timer) clearTimeout(timer);
  }
}

export const get = <T = unknown>(url: string, options: Omit<HttpOptions, 'method'> = {}) => fetch<T>(url, { ...options, method: 'GET' });
export const post = <T = unknown>(url: string, body?: unknown, options: Omit<HttpOptions, 'method' | 'body'> = {}) => fetch<T>(url, { ...options, method: 'POST', body });
export const put = <T = unknown>(url: string, body?: unknown, options: Omit<HttpOptions, 'method' | 'body'> = {}) => fetch<T>(url, { ...options, method: 'PUT', body });
export const del = <T = unknown>(url: string, options: Omit<HttpOptions, 'method'> = {}) => fetch<T>(url, { ...options, method: 'DELETE' });
export { fetch as tauriFetch };
