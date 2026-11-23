import { useMemo, useState } from 'react'
import {
  Alert,
  Avatar,
  Badge,
  Box,
  Button,
  CircularProgress,
  IconButton,
  InputAdornment,
  List,
  ListItemAvatar,
  ListItemButton,
  ListItemText,
  TextField,
  Typography,
} from '@mui/material'

import AddRoundedIcon from '@mui/icons-material/AddRounded'
import SearchRoundedIcon from '@mui/icons-material/SearchRounded'

export default function ConversationSidebar({
  conversations,
  selectedID,
  onSelect,
  loading = false,
  error = '',
  onRetry,
}) {
  const [query, setQuery] = useState('')

  const filteredConversations = useMemo(() => {
    const keyword = query.trim().toLowerCase()

    if (!keyword) {
      return conversations
    }

    return conversations.filter((conversation) =>
      conversation.name.toLowerCase().includes(keyword),
    )
  }, [conversations, query])

  return (
    <Box
      sx={{
        width: 330,
        flexShrink: 0,
        bgcolor: '#f7f8fa',
        borderRight: '1px solid #e5e8ed',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <Box
        sx={{
          minHeight: 66,
          px: 2,
          py: 1.5,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <Typography
          sx={{
            fontSize: 20,
            fontWeight: 700,
            letterSpacing: '-0.3px',
          }}
        >
          消息
        </Typography>

        <IconButton size="small">
          <AddRoundedIcon />
        </IconButton>
      </Box>

      <Box sx={{ px: 2, pb: 1.5 }}>
        <TextField
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="搜索会话"
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchRoundedIcon
                  sx={{
                    color: '#98a1af',
                    fontSize: 19,
                  }}
                />
              </InputAdornment>
            ),
          }}
          sx={{
            '& .MuiOutlinedInput-root': {
              height: 36,
              bgcolor: '#eceff3',
              borderRadius: 2,
              '& fieldset': {
                border: 'none',
              },
            },
            '& input': {
              fontSize: 13,
            },
          }}
        />
      </Box>

      {error && (
        <Alert
          severity="error"
          action={
            <Button
              color="inherit"
              size="small"
              onClick={onRetry}
            >
              重试
            </Button>
          }
          sx={{ mx: 2, mb: 1 }}
        >
          {error}
        </Alert>
      )}

      <List
        disablePadding
        className="chat-scroll"
        sx={{
          flex: 1,
          overflowY: 'auto',
          px: 1,
          pb: 1,
        }}
      >
        {loading && conversations.length === 0 && (
          <Box
            sx={{
              minHeight: 220,
              display: 'grid',
              placeItems: 'center',
            }}
          >
            <CircularProgress size={28} />
          </Box>
        )}

        {!loading && filteredConversations.length === 0 && (
          <Box
            sx={{
              px: 2,
              py: 5,
              textAlign: 'center',
            }}
          >
            <Typography
              sx={{
                fontSize: 13,
                color: '#8c95a3',
              }}
            >
              {query ? '没有匹配的会话' : '暂无会话'}
            </Typography>
          </Box>
        )}

        {filteredConversations.map((conversation) => (
          <ListItemButton
            key={conversation.id}
            selected={selectedID === conversation.id}
            onClick={() => onSelect(conversation.id)}
            sx={{
              minHeight: 72,
              mb: 0.5,
              px: 1.5,
              borderRadius: 2,
              '&.Mui-selected': {
                bgcolor: '#e8eefb',
              },
              '&.Mui-selected:hover': {
                bgcolor: '#e3eafa',
              },
              '&:hover': {
                bgcolor: '#eef1f5',
              },
            }}
          >
            <ListItemAvatar>
              <Badge
                badgeContent={conversation.unread}
                color="primary"
                max={99}
                overlap="circular"
              >
                <Avatar
                  src={conversation.avatarUrl || undefined}
                  sx={{
                    width: 44,
                    height: 44,
                    bgcolor: conversation.color,
                    fontSize: 16,
                    fontWeight: 700,
                  }}
                >
                  {conversation.name.slice(0, 1)}
                </Avatar>
              </Badge>
            </ListItemAvatar>

            <ListItemText
              primary={conversation.name}
              secondary={conversation.preview}
              primaryTypographyProps={{
                fontSize: 14,
                fontWeight: 600,
                noWrap: true,
              }}
              secondaryTypographyProps={{
                fontSize: 12,
                mt: 0.3,
                noWrap: true,
              }}
              sx={{ ml: 1, mr: 1 }}
            />

            <Typography
              variant="caption"
              sx={{
                alignSelf: 'flex-start',
                mt: 0.6,
                color: '#9aa3b1',
                fontSize: 11,
                whiteSpace: 'nowrap',
              }}
            >
              {conversation.time}
            </Typography>
          </ListItemButton>
        ))}
      </List>
    </Box>
  )
}