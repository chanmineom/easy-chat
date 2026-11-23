import { useAuthStore } from '../stores/authStore'

export class ApiError extends Error {
  constructor(message, status = 0, code = 0) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

export async function request(path, options = {}) {
  const state = useAuthStore.getState()
  const headers = new Headers(options.headers || {})

  let body = options.body

  if (
    body !== undefined &&
    body !== null &&
    !(body instanceof FormData)
  ) {
    headers.set('Content-Type', 'application/json')
    body = JSON.stringify(body)
  }

  if (state.token) {
    headers.set('Authorization', `Bearer ${state.token}`)
  }

  let response

  try {
    response = await fetch(path, {
      ...options,
      headers,
      body,
    })
  } catch {
    throw new ApiError('网络连接失败')
  }

  const text = await response.text()
  let payload = null

  if (text) {
    try {
      payload = JSON.parse(text)
    } catch {
      throw new ApiError('服务器返回了无法解析的响应', response.status)
    }
  }

  if (response.status === 401) {
    useAuthStore.getState().logout()
    throw new ApiError(
      payload?.msg || '登录已失效，请重新登录',
      response.status,
      payload?.code || 401,
    )
  }

  if (!response.ok) {
    throw new ApiError(
      payload?.msg || `请求失败：${response.status}`,
      response.status,
      payload?.code,
    )
  }

  if (payload && payload.code >= 400) {
    throw new ApiError(
      payload.msg || '请求失败',
      response.status,
      payload.code,
    )
  }

  return payload?.data
}