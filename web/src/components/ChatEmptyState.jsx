import {
  Box,
  Divider,
  IconButton,
  TextField,
  Typography,
} from '@mui/material'

import ChatBubbleOutlineRoundedIcon from '@mui/icons-material/ChatBubbleOutlineRounded'
import MoreHorizRoundedIcon from '@mui/icons-material/MoreHorizRounded'
import SearchRoundedIcon from '@mui/icons-material/SearchRounded'
import SentimentSatisfiedAltRoundedIcon from '@mui/icons-material/SentimentSatisfiedAltRounded'
import AddCircleOutlineRoundedIcon from '@mui/icons-material/AddCircleOutlineRounded'

export default function ChatEmptyState({ conversation }) {
  return (
    <Box
      component="main"
      sx={{
        flex: 1,
        minWidth: 0,
        display: 'flex',
        flexDirection: 'column',
        bgcolor: '#f4f6f9',
      }}
    >
      <Box
        sx={{
          minHeight: 66,
          px: 3,
          display: 'flex',
          alignItems: 'center',
          borderBottom: '1px solid #e5e8ed',
          bgcolor: 'rgba(255,255,255,0.72)',
          backdropFilter: 'blur(12px)',
        }}
      >
        <Box>
          <Typography
            sx={{
              fontSize: 16,
              fontWeight: 700,
            }}
          >
            {conversation?.name || '请选择会话'}
          </Typography>

          <Typography
            sx={{
              fontSize: 11,
              color: '#8c95a3',
              mt: 0.2,
            }}
          >
            在线
          </Typography>
        </Box>

        <Box sx={{ flex: 1 }} />

        <IconButton size="small">
          <SearchRoundedIcon />
        </IconButton>

        <IconButton size="small">
          <MoreHorizRoundedIcon />
        </IconButton>
      </Box>

      <Box
        sx={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <Box
          sx={{
            width: 96,
            height: 96,
            borderRadius: '50%',
            display: 'grid',
            placeItems: 'center',
            bgcolor: '#e7ebf2',
            mb: 2.5,
          }}
        >
          <ChatBubbleOutlineRoundedIcon
            sx={{
              fontSize: 48,
              color: '#aab3c0',
            }}
          />
        </Box>

        <Typography
          sx={{
            fontSize: 16,
            fontWeight: 600,
          }}
        >
          选择一个会话开始聊天
        </Typography>

        <Typography
          sx={{
            mt: 0.8,
            color: '#8c95a3',
            fontSize: 13,
          }}
        >
          消息功能将在下一节接入
        </Typography>
      </Box>

      <Divider />

      <Box
        sx={{
          minHeight: 164,
          p: 2,
          display: 'flex',
          flexDirection: 'column',
          bgcolor: '#f4f6f9',
        }}
      >
        <Box sx={{ display: 'flex', gap: 0.5 }}>
          <IconButton size="small">
            <SentimentSatisfiedAltRoundedIcon />
          </IconButton>

          <IconButton size="small">
            <AddCircleOutlineRoundedIcon />
          </IconButton>
        </Box>

        <TextField
          multiline
          minRows={2}
          maxRows={4}
          placeholder="输入消息"
          sx={{
            mt: 1,
            flex: 1,
            '& .MuiOutlinedInput-root': {
              bgcolor: '#ffffff',
              alignItems: 'flex-start',
            },
          }}
        />
      </Box>
    </Box>
  )
}