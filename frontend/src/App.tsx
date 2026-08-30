import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import type { ReactElement } from 'react'
import { useAuthStore } from './store/auth'
import AppLayout from './layouts/AppLayout'
import Login from './pages/Login'
import Register from './pages/Register'
import Markets from './pages/Markets'
import Spot from './pages/Spot'
import Futures from './pages/Futures'
import Assets from './pages/Assets'
import Learn from './pages/Learn'

function RequireAuth({ children }: { children: ReactElement }) {
  const token = useAuthStore((s) => s.token)
  if (!token) return <Navigate to="/login" replace />
  return children
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route
          path="/"
          element={
            <RequireAuth>
              <AppLayout />
            </RequireAuth>
          }
        >
          <Route index element={<Navigate to="/markets" replace />} />
          <Route path="markets" element={<Markets />} />
          <Route path="spot" element={<Spot />} />
          <Route path="futures" element={<Futures />} />
          <Route path="assets" element={<Assets />} />
          <Route path="learn" element={<Learn />} />
        </Route>
        <Route path="*" element={<Navigate to="/markets" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
