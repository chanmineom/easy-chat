import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export const useAuthStore = create(
  persist(
    (set) => ({
      token: '',
      expire: 0,
      user: null,

      setSession: ({ token, expire }) => {
        set({
          token,
          expire,
          user: null,
        })
      },

      setUser: (user) => {
        set({ user })
      },

      logout: () => {
        set({
          token: '',
          expire: 0,
          user: null,
        })
      },
    }),
    {
      name: 'easy-chat-auth',
      partialize: (state) => ({
        token: state.token,
        expire: state.expire,
        user: state.user,
      }),
    },
  ),
)

