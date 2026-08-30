import { create } from 'zustand'
import type { User } from '../api/types'

interface AuthState {
  token: string
  apiKey: string
  apiSecret: string
  user: User | null
  setAuth: (token: string, apiKey: string, apiSecret: string, user: User) => void
  setCredentials: (apiKey: string, apiSecret: string) => void
  setUser: (user: User) => void
  logout: () => void
}

interface Saved {
  token: string
  apiKey: string
  apiSecret: string
  user: User
}

const saved = JSON.parse(localStorage.getItem('cryptosim.auth') || 'null') as Saved | null

export const useAuthStore = create<AuthState>((set) => ({
  token: saved?.token || '',
  apiKey: saved?.apiKey || '',
  apiSecret: saved?.apiSecret || '',
  user: saved?.user || null,
  setAuth: (token, apiKey, apiSecret, user) => {
    localStorage.setItem('cryptosim.auth', JSON.stringify({ token, apiKey, apiSecret, user }))
    set({ token, apiKey, apiSecret, user })
  },
  setCredentials: (apiKey, apiSecret) => {
    const s = useAuthStore.getState()
    localStorage.setItem(
      'cryptosim.auth',
      JSON.stringify({ token: s.token, apiKey, apiSecret, user: s.user }),
    )
    set({ apiKey, apiSecret })
  },
  setUser: (user) => {
    const s = useAuthStore.getState()
    localStorage.setItem(
      'cryptosim.auth',
      JSON.stringify({ token: s.token, apiKey: s.apiKey, apiSecret: s.apiSecret, user }),
    )
    set({ user })
  },
  logout: () => {
    localStorage.removeItem('cryptosim.auth')
    set({ token: '', apiKey: '', apiSecret: '', user: null })
  },
}))
