// 轻量 i18n：中/英字典 + 全局语言状态（localStorage 记忆）。
// 覆盖核心文案；业务长文本（学习中心内容）保持中文原文。
import { create } from 'zustand'

export type Lang = 'zh' | 'en'

const dict = {
  nav: { zh: '行情', en: 'Markets' },
  navSpot: { zh: '现货交易', en: 'Spot' },
  navFutures: { zh: '合约交易', en: 'Futures' },
  navAssets: { zh: '资产', en: 'Assets' },
  navLearn: { zh: '学习中心', en: 'Learn' },
  login: { zh: '登录', en: 'Sign in' },
  register: { zh: '注册', en: 'Sign up' },
  logout: { zh: '退出登录', en: 'Sign out' },
  resetAccount: { zh: '重置账户', en: 'Reset account' },
  resetCred: { zh: '重置 API 凭证', en: 'Reset API keys' },
  myAssets: { zh: '我的资产（虚拟）', en: 'My Assets (virtual)' },
  available: { zh: '可用余额', en: 'Available' },
  frozen: { zh: '冻结', en: 'Frozen' },
  refresh: { zh: '刷新', en: 'Refresh' },
  appTitle: { zh: 'CryptoSim — 虚拟加密货币交易所（模拟盘）', en: 'CryptoSim — Virtual Crypto Exchange (Paper Trading)' },
  disclaimer: { zh: '资金全虚拟 · 仅供学习', en: 'Virtual funds only · For learning' },
} as const

export type DictKey = keyof typeof dict

interface LangState {
  lang: Lang
  toggle: () => void
}

const saved = (localStorage.getItem('cryptosim.lang') as Lang) || 'zh'

export const useLang = create<LangState>((set) => ({
  lang: saved,
  toggle: () => {
    const next: Lang = useLang.getState().lang === 'zh' ? 'en' : 'zh'
    localStorage.setItem('cryptosim.lang', next)
    set({ lang: next })
  },
}))

/** t('nav') → 当前语言文案 */
export function t(key: DictKey, lang: Lang): string {
  return dict[key][lang]
}
