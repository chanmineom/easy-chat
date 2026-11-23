import { request } from './http'

export async function login({ phone, password }) {
  return request('/v1/user/login', {
    method: 'POST',
    body: {
      phone,
      password,
    },
  })
}

export async function register({
  phone,
  password,
  nickname,
}) {
  return request('/v1/user/register', {
    method: 'POST',
    body: {
      phone,
      password,
      nickname,
    },
  })
}

export async function getCurrentUser() {
  return request('/v1/user/detail')
}