import {
  Avatar,
  Box,
  IconButton,
  Tooltip,
} from '@mui/material'

import ChatBubbleOutlineRoundedIcon from '@mui/icons-material/ChatBubbleOutlineRounded'
import ContactsOutlinedIcon from '@mui/icons-material/ContactsOutlined'
import GroupsOutlinedIcon from '@mui/icons-material/GroupsOutlined'
import LogoutRoundedIcon from '@mui/icons-material/LogoutRounded'

import { useAuthStore } from '../stores/authStore'

const items = [
  {
    key: 'chat',
    label: '消息',
    icon: <ChatBubbleOutlineRoundedIcon />,
  },
  {
    key: 'contacts',
    label: '联系人',
    icon: <ContactsOutlinedIcon />,
  },
  {
    key: 'groups',
    label: '群组',
    icon: <GroupsOutlinedIcon />,
  },
]

export default function NavigationRail({
  active,
  onChange,
}) {
  const user = useAuthStore((state) => state.user)
  const logout = useAuthStore((state) => state.logout)

  return (
    <Box
      component="nav"
      sx={{
        width: 68,
        flexShrink: 0,
        bgcolor: '#20242b',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        py: 2,
      }}
    >
      <Avatar
        src={user?.avatar_url || undefined}
        sx={{
          width: 40,
          height: 40,
          mb: 3,
          bgcolor: '#2f6fed',
          fontSize: 15,
          fontWeight: 700,
        }}
      >
        {user?.nickname?.slice(0, 1) || 'U'}
      </Avatar>

      {items.map((item) => {
        const isActive = active === item.key

        return (
          <Tooltip
            key={item.key}
            title={item.label}
            placement="right"
          >
            <IconButton
              onClick={() => onChange(item.key)}
              sx={{
                width: 48,
                height: 48,
                mb: 1,
                color: isActive
                  ? '#ffffff'
                  : 'rgba(255,255,255,0.55)',
                bgcolor: isActive
                  ? 'rgba(47,111,237,0.9)'
                  : 'transparent',
                '&:hover': {
                  color: '#ffffff',
                  bgcolor: 'rgba(255,255,255,0.1)',
                },
              }}
            >
              {item.icon}
            </IconButton>
          </Tooltip>
        )
      })}

      <Box sx={{ flex: 1 }} />

      <Tooltip title="退出登录" placement="right">
        <IconButton
          onClick={logout}
          sx={{
            color: 'rgba(255,255,255,0.55)',
            '&:hover': {
              color: '#ffffff',
              bgcolor: 'rgba(255,255,255,0.1)',
            },
          }}
        >
          <LogoutRoundedIcon />
        </IconButton>
      </Tooltip>
    </Box>
  )
}