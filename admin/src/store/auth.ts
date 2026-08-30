import { create } from 'zustand'

interface AdminState {
  token: string
  apiKey: string
  apiSecret: string
  username: string
  setAuth: (token: string, apiKey: string, apiSecret: string, username: string) => void
  logout: () => void
}

const saved = JSON.parse(localStorage.getItem('cryptosim.admin') || 'null') as {
  token: string
  apiKey: string
  apiSecret: string
  username: string
} | null

export const useAdminStore = create<AdminState>((set) => ({
  token: saved?.token || '',
  apiKey: saved?.apiKey || '',
  apiSecret: saved?.apiSecret || '',
  username: saved?.username || '',
  setAuth: (token, apiKey, apiSecret, username) => {
    localStorage.setItem('cryptosim.admin', JSON.stringify({ token, apiKey, apiSecret, username }))
    set({ token, apiKey, apiSecret, username })
  },
  logout: () => {
    localStorage.removeItem('cryptosim.admin')
    set({ token: '', apiKey: '', apiSecret: '', username: '' })
  },
}))
