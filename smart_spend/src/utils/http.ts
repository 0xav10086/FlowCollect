/**
 * HTTP 请求工具模块
 * 基于 VITE_API_BASE_URL 动态构建请求地址，消除硬编码路径
 * 当 VITE_API_BASE_URL 为空时，使用相对路径（适用于 Nginx 反代同域部署）
 */

const BASE_URL = (import.meta.env.VITE_API_BASE_URL as string) || ''

/**
 * 构建完整的 API URL
 * @param path - API 路径，如 '/api/stats'
 */
export function buildApiUrl(path: string): string {
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  if (!BASE_URL) {
    return normalizedPath
  }
  const normalizedBase = BASE_URL.replace(/\/+$/, '')
  return `${normalizedBase}${normalizedPath}`
}

/**
 * 将 HTTP(S) URL 转换为 WS(S) URL
 * @param baseUrl - 如 'https://api-dash.0xav10086.space' 或 'http://localhost:8080'
 */
export function toWsUrl(baseUrl: string): string {
  return baseUrl.replace(/^http/, 'ws')
}

/**
 * 封装 fetch，自动拼接 BASE_URL 和 Authorization header
 * @param path - API 路径，如 '/api/stats'
 * @param options - fetch RequestInit 扩展
 */
export async function apiFetch<T = any>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const url = buildApiUrl(path)
  const token = localStorage.getItem('token')

  const headers: Record<string, string> = {
    ...(options.headers as Record<string, string>),
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const res = await fetch(url, { ...options, headers })

  if (!res.ok) {
    throw new Error(`API request failed: ${res.status} ${res.statusText}`)
  }

  return res.json()
}

/**
 * 封装 POST 请求（JSON body）
 * @param path - API 路径
 * @param body - 请求体对象
 */
export async function apiPost<T = any>(
  path: string,
  body: Record<string, any>
): Promise<T> {
  return apiFetch<T>(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}
