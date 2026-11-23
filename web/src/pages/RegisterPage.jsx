import { useState } from 'react'
import {
  Alert,
  Button,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import { Link, useNavigate } from 'react-router-dom'

import { register } from '../api/auth'
import { useAuthStore } from '../stores/authStore'

export default function RegisterPage() {
  const navigate = useNavigate()
  const setSession = useAuthStore((state) => state.setSession)

  const [form, setForm] = useState({
    phone: '',
    nickname: '',
    password: '',
    confirmPassword: '',
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

    if (form.password !== form.confirmPassword) {
      setError('两次输入的密码不一致')
      return
    }

    setLoading(true)

    try {
      const data = await register({
        phone: form.phone.trim(),
        password: form.password,
        nickname: form.nickname.trim(),
      })

      setSession({
        token: data.token,
        expire: data.expire,
      })

      navigate('/app', { replace: true })
    } catch (err) {
      setError(err.message || '注册失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      <Stack spacing={2.2}>
        <div>
          <Typography
            sx={{
              fontSize: 26,
              fontWeight: 800,
              letterSpacing: '-0.8px',
            }}
          >
            创建账号
          </Typography>
          <Typography
            sx={{
              mt: 0.8,
              color: '#8c95a3',
              fontSize: 13,
            }}
          >
            注册后即可添加好友和发送消息
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
          label="昵称"
          name="nickname"
          value={form.nickname}
          onChange={updateField}
          autoComplete="nickname"
        />

        <TextField
          label="密码"
          name="password"
          type="password"
          value={form.password}
          onChange={updateField}
          autoComplete="new-password"
          helperText="至少 8 个字符"
        />

        <TextField
          label="确认密码"
          name="confirmPassword"
          type="password"
          value={form.confirmPassword}
          onChange={updateField}
          autoComplete="new-password"
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
          {loading ? '注册中...' : '注册并进入'}
        </Button>

        <Typography
          sx={{
            textAlign: 'center',
            fontSize: 13,
            color: '#7b8494',
          }}
        >
          已有账号？
          <Link
            to="/login"
            style={{
              color: '#2f6fed',
              marginLeft: 4,
              fontWeight: 600,
              textDecoration: 'none',
            }}
          >
            登录
          </Link>
        </Typography>
      </Stack>
    </form>
  )
}