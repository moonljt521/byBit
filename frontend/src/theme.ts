// 交易所暗色设计体系：全局唯一色彩来源。
export const C = {
  bg: '#0B0E11',
  surface: '#161A1E',
  surface2: '#1E2329',
  border: '#252930',
  text: '#EAECEF',
  text2: '#868E93',
  up: '#0ECB81',
  down: '#F6465D',
  accent: '#00C8A0',
} as const

/** 行情 sparkline：把收盘价序列画成折线路径（返回 SVG points 字符串）。 */
export function sparkPoints(closes: number[], w: number, h: number): string {
  if (closes.length < 2) return ''
  const min = Math.min(...closes)
  const max = Math.max(...closes)
  const span = max - min || 1
  return closes
    .map((v, i) => {
      const x = (i / (closes.length - 1)) * w
      const y = h - ((v - min) / span) * h
      return `${x.toFixed(1)},${y.toFixed(1)}`
    })
    .join(' ')
}
