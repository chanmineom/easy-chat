import { useEffect, useState } from 'react'
import {
  Box,
  CircularProgress,
} from '@mui/material'
import {
  Navigate,
  Outlet,
  useLocation,
} from 'react-router-dom'

import { getCurrentUser } from '../api/auth'
import { useAuthStore } from '../stores/authStore'

export default function ProtectedRoute() {
  const location = useLocation()

  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const setUser = useAuthStore((state) => state.setUser)
  const logout = useAuthStore((state) => state.logout)

  const [loading, setLoading] = useState(
    Boolean(token && !user),
  )

  useEffect(() => {
    if (!token) {
      setLoading(false)
      return
    }

    if (user) {
      setLoading(false)
      return
    }

    let cancelled = false

    async function loadUser() {
      try {
        const data = await getCurrentUser()

        if (!cancelled) {
          setUser(data.info)
        }
      } catch {
        if (!cancelled) {
          logout()
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    loadUser()

    return () => {
      cancelled = true
    }
  }, [token, user, setUser, logout])

  if (!token) {
    return (
      <Navigate
        to="/login"
        replace
        state={{ from: location.pathname }}
      />
    )
  }

  if (loading) {
    return (
      <Box
        sx={{
          width: '100%',
          height: '100%',
          display: 'grid',
          placeItems: 'center',
          bgcolor: '#eef1f5',
        }}
      >
        <CircularProgress />
      </Box>
    )
  }

  return <Outlet />
}