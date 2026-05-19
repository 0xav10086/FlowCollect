/**
 * WebSocket 连接工具模块
 * 基于 VITE_API_BASE_URL 自动推导 WS/WSS 协议
 */

import { toWsUrl, buildApiUrl } from './http'

export interface WsOptions {
  /** API 路径，如 '/ws/traffic' */
  path: string
  /** 连接成功回调 */
  onOpen?: (ws: WebSocket) => void
  /** 消息回调 */
  onMessage?: (data: any, ws: WebSocket) => void
  /** 连接关闭回调 */
  onClose?: (event: CloseEvent) => void
  /** 错误回调 */
  onError?: (event: Event) => void
  /** 自动重连间隔 (ms)，默认 3000，设为 0 禁用 */
  reconnectInterval?: number
}

/**
 * 构建 WebSocket URL
 * 将 VITE_API_BASE_URL 的 http/https 替换为 ws/wss
 * @param path - WS 路径，如 '/ws/traffic'
 */
export function buildWsUrl(path: string): string {
  const baseUrl = import.meta.env.VITE_API_BASE_URL as string
  const wsBase = toWsUrl(baseUrl)
  const normalizedBase = wsBase.replace(/\/+$/, '')
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  return `${normalizedBase}${normalizedPath}`
}

/**
 * 创建 WebSocket 连接，支持自动重连
 * @param options - 连接配置
 * @returns 关闭函数
 */
export function createWebSocket(options: WsOptions): () => void {
  const {
    path,
    onOpen,
    onMessage,
    onClose,
    onError,
    reconnectInterval = 3000,
  } = options

  const url = buildWsUrl(path)
  let ws: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let manuallyClosed = false

  const connect = () => {
    ws = new WebSocket(url)

    ws.onopen = () => {
      console.log(`[WS] Connected: ${url}`)
      onOpen?.(ws!)
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        onMessage?.(data, ws!)
      } catch {
        onMessage?.(event.data, ws!)
      }
    }

    ws.onclose = (event) => {
      console.log(`[WS] Closed: ${url} (code: ${event.code})`)
      onClose?.(event)

      if (!manuallyClosed && reconnectInterval > 0) {
        console.log(`[WS] Reconnecting in ${reconnectInterval}ms...`)
        reconnectTimer = setTimeout(connect, reconnectInterval)
      }
    }

    ws.onerror = (event) => {
      console.error(`[WS] Error: ${url}`)
      onError?.(event)
    }
  }

  connect()

  // 返回关闭函数
  return () => {
    manuallyClosed = true
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    if (ws) {
      ws.close()
      ws = null
    }
  }
}
