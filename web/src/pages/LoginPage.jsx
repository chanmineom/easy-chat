import { useState } from 'react'
import {
  Alert,
  Button,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import {
  Link,
  useLocation,
  useNavigate,
} from 'react-router-dom'

import { login } from '../api/auth'
import { useAuthStore } from '../stores/authStore'

export default function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const setSession = useAuthStore((state) => state.setSession)

  const [form, setForm] = useState({
    phone: '',
    password: '',
  })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  function updateField(event) {
    const { name, value } = event.target

    setForm((current) => ({
      ...current,
      [name]: value,
    }))
  }

  async function handleSubmit(event) {
    event.preventDefault()
    setError('')
    setLoading(true)

    try {
      const data = await login({
        phone: form.phone.trim(),
        password: form.password,
      })

      setSession({
        token: data.token,
        expire: data.expire,
      })

      const nextPath =
        location.state?.from || '/app'

      navigate(nextPath, { replace: true })
    } catch (err) {
      setError(err.message || '登录失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      <Stack spacing={2.5}>
        <div>
          <Typography
            sx={{
              fontSize: 26,
              fontWeight: 800,
              letterSpacing: '-0.8px',
            }}
          >
            登录
          </Typography>
          <Typography
            sx={{
              mt: 0.8,
              color: '#8c95a3',
              fontSize: 13,
            }}
          >
            登录后继续查看你的消息
          </Typography>
        </div>

        {error && (
          <Alert severity="error">
            {error}
          </Alert>
        )}

        <TextField
          label="手机号"
          name="phone"
          value={form.phone}
          onChange={updateField}
          autoComplete="tel"
          autoFocus
        />

        <TextField
          label="密码"
          name="password"
          type="password"
          value={form.password}
          onChange={updateField}
          autoComplete="current-password"
        />

        <Button
          type="submit"
          variant="contained"
          size="large"
          disabled={loading}
          sx={{
            height: 46,
            fontSize: 15,
          }}
        >
          {loading ? '登录中...' : '登录'}
        </Button>

        <Typography
          sx={{
            textAlign: 'center',
            fontSize: 13,
            color: '#7b8494',
          }}
        >
          还没有账号？
          <Link
            to="/register"
            style={{
              color: '#2f6fed',
              marginLeft: 4,
              fontWeight: 600,
              textDecoration: 'none',
            }}
          >
            注册
          </Link>
        </Typography>
      </Stack>
    </form>
  )
}