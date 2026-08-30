import { useEffect, useRef, useState } from 'react'
import { init, dispose } from 'klinecharts'
import type { Chart } from 'klinecharts'
import { Button, Space } from 'antd'
import { fetchKlines } from '../api/market'

const BARS = ['1m', '5m', '15m', '1h', '4h', '1d'] as const
export type Bar = (typeof BARS)[number]

interface Props {
  symbol: string
  height?: number
}

/** 专业 K 线图（klinecharts），15s 轮询刷新。 */
export default function CandleChart({ symbol, height = 460 }: Props) {
  const boxRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<Chart | null>(null)
  const [bar, setBar] = useState<Bar>('1m')

  useEffect(() => {
    if (!boxRef.current) return
    const chart = init(boxRef.current, {
      styles: {
        grid: {
          horizontal: { color: '#252930' },
          vertical: { color: '#252930' },
        },
        candle: {
          bar: {
            upColor: '#0ECB81',
            downColor: '#F6465D',
            noChangeColor: '#868E93',
          },
        },
        indicator: {
          bars: [{ upColor: '#0ECB81', downColor: '#F6465D' }],
        },
      },
    })
    chart?.createIndicator('MA', false, { id: 'candle_pane' })
    chart?.createIndicator('VOL')
    chartRef.current = chart
    const dom = boxRef.current
    return () => {
      dispose(dom)
      chartRef.current = null
    }
  }, [])

  useEffect(() => {
    let stop = false
    async function load() {
      try {
        const { candles } = await fetchKlines(symbol, bar, 200)
        if (stop || !chartRef.current) return
        chartRef.current.applyNewData(
          candles.map((c) => ({
            timestamp: c.ts,
            open: +c.o,
            high: +c.h,
            low: +c.l,
            close: +c.c,
            volume: +c.vol,
          })),
        )
      } catch {
        /* 拦截器已提示 */
      }
    }
    void load()
    const timer = window.setInterval(load, 15000)
    return () => {
      stop = true
      window.clearInterval(timer)
    }
  }, [symbol, bar])

  return (
    <div>
      <Space style={{ marginBottom: 8 }}>
        {BARS.map((b) => (
          <Button key={b} size="small" type={b === bar ? 'primary' : 'text'} onClick={() => setBar(b)}>
            {b.toUpperCase()}
          </Button>
        ))}
      </Space>
      <div ref={boxRef} style={{ height }} />
    </div>
  )
}
