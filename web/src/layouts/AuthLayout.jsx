import { Box, Paper, Typography } from '@mui/material'
import { Outlet } from 'react-router-dom'

import ChatBubbleOutlineRoundedIcon from '@mui/icons-material/ChatBubbleOutlineRounded'

export default function AuthLayout() {
  return (
    <Box
      sx={{
        width: '100%',
        minHeight: '100%',
        display: 'grid',
        placeItems: 'center',
        px: 2,
        py: 4,
        background:
          'radial-gradient(circle at 15% 20%, rgba(47,111,237,0.15), transparent 32%), radial-gradient(circle at 85% 80%, rgba(42,157,143,0.12), transparent 30%), #eef1f5',
      }}
    >
      <Paper
        elevation={0}
        sx={{
          width: '100%',
          maxWidth: 430,
          p: { xs: 3, sm: 4.5 },
          border: '1px solid rgba(255,255,255,0.8)',
          boxShadow: '0 24px 80px rgba(31,41,55,0.12)',
          bgcolor: 'rgba(255,255,255,0.94)',
          backdropFilter: 'blur(20px)',
        }}
      >
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: 1.2,
            mb: 4,
          }}
        >
          <Box
            sx={{
              width: 42,
              height: 42,
              borderRadius: 2,
              bgcolor: '#2f6fed',
              display: 'grid',
              placeItems: 'center',
              color: '#ffffff',
            }}
          >
            <ChatBubbleOutlineRoundedIcon />
          </Box>

          <Box>
            <Typography
              sx={{
                fontSize: 18,
                fontWeight: 800,
                letterSpacing: '-0.3px',
              }}
            >
              Easy Chat
            </Typography>
            <Typography
              sx={{
                fontSize: 11,
                color: '#8c95a3',
              }}
            >
              Distributed IM
            </Typography>
          </Box>
        </Box>

        <Outlet />
      </Paper>
    </Box>
  )
}