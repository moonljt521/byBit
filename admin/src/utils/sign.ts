// HMAC 请求签名（与后端 middleware.Private 规范一致）：
//   stringToSign = timestamp \n METHOD \n path \n query \n sha256hex(body)
//   signature = hex(HMAC-SHA256(apiSecret, stringToSign))
//   query 按 key 字典序排列（与 axios 序列化顺序一致）

async function sha256Hex(input: string): Promise<string> {
  const buf = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(input))
  return [...new Uint8Array(buf)].map((b) => b.toString(16).padStart(2, '0')).join('')
}

async function hmacHex(secret: string, message: string): Promise<string> {
  const key = await crypto.subtle.importKey(
    'raw',
    new TextEncoder().encode(secret),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign'],
  )
  const buf = await crypto.subtle.sign('HMAC', key, new TextEncoder().encode(message))
  return [...new Uint8Array(buf)].map((b) => b.toString(16).padStart(2, '0')).join('')
}

export interface SignInput {
  method: string
  path: string
  query: Record<string, unknown> | undefined
  body: string
  apiSecret: string
}

export async function signRequest(input: SignInput): Promise<{
  timestamp: string
  signature: string
}> {
  const timestamp = String(Math.floor(Date.now() / 1000))
  const sortedQuery = input.query
    ? Object.keys(input.query)
        .filter((k) => input.query![k] !== undefined && input.query![k] !== null && input.query![k] !== '')
        .sort()
        .map((k) => `${k}=${encodeURIComponent(String(input.query![k]))}`)
        .join('&')
    : ''
  const bodyHash = await sha256Hex(input.body)
  const sts = [timestamp, input.method.toUpperCase(), input.path, sortedQuery, bodyHash].join('\n')
  const signature = await hmacHex(input.apiSecret, sts)
  return { timestamp, signature }
}

/** 按字典序构造 query 串（与签名计算保持一致）。 */
export function sortedQuery(params: Record<string, unknown> | undefined): Record<string, unknown> | undefined {
  if (!params) return undefined
  const out: Record<string, unknown> = {}
  for (const k of Object.keys(params).sort()) {
    const v = params[k]
    if (v !== undefined && v !== null && v !== '') out[k] = v
  }
  return out
}
