import {
  useEffect,
  useRef,
  useState,
} from 'react'
import {
  Alert,
  Avatar,
  Box,
  Button,
  CircularProgress,
  Divider,
  IconButton,
  Paper,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material'

import MoreHorizRoundedIcon from '@mui/icons-material/MoreHorizRounded'
import RefreshRoundedIcon from '@mui/icons-material/RefreshRounded'
import SendRoundedIcon from '@mui/icons-material/SendRounded'

import { syncMessages } from '../api/im'
import { listGroupMembers } from '../api/social'
import { getUsers } from '../api/users'
import {
  chatSocket,
  createID,
} from '../realtime/chatSocket'
import { useAuthStore } from '../stores/authStore'

import {
  cacheMessages,
  getSyncCursor,
  listPendingMessages,
  loadCachedMessages,
  queuePendingMessage,
  removePendingMessage,
  saveSyncCursor,
  updatePendingMessage,
} from '../storage/chatCache'

const PAGE_SIZE = 50

function formatMessageTime(timestamp) {
  if (!timestamp) {
    return ''
  }

  return new Intl.DateTimeFormat('zh-CN', {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(timestamp))
}

function sortMessages(messages) {
  return [...messages].sort((left, right) => {
    if (left.sequence > 0 && right.sequence > 0) {
      return left.sequence - right.sequence
    }

    if (left.sequence > 0) {
      return 1
    }

    if (right.sequence > 0) {
      return -1
    }

    return left.created_at - right.created_at
  })
}

export default function MessagePanel({
  conversation,
  socketStatus = 'idle',
}) {
  const currentUser = useAuthStore((state) => state.user)

  const [messages, setMessages] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [sendError, setSendError] = useState('')
  const [input, setInput] = useState('')
  const [reloadToken, setReloadToken] = useState(0)
  const [senderProfiles, setSenderProfiles] = useState({})
  const [hasOlder, setHasOlder] = useState(false)
  const [loadingOlder, setLoadingOlder] = useState(false)
  const [olderError, setOlderError] = useState('')

  const scrollContainerRef = useRef(null)
  const scrollContentRef = useRef(null)
  const suppressAutoScrollRef = useRef(false)

  function scrollToLatest() {
    if (suppressAutoScrollRef.current) {
      return 0
    }

    const frame = window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => {
        if (suppressAutoScrollRef.current) {
          return
        }

        const container = scrollContainerRef.current

        if (!container) {
          return
        }

        container.scrollTop = container.scrollHeight
      })
    })

    return frame
  }

  useEffect(() => {
    if (
      conversation?.chatType !== 1 ||
      !conversation?.id
    ) {
      setSenderProfiles({})
      return undefined
    }

    let cancelled = false

    async function loadSenderProfiles() {
      try {
        const memberData = await listGroupMembers(
          conversation.id,
        )
        const memberIDs = (memberData.list || []).map(
          (member) => member.user_id,
        )
        const users = await getUsers(memberIDs)
        const profiles = Object.fromEntries(
          users.map((user) => [user.id, user]),
        )

        if (currentUser?.id) {
          profiles[currentUser.id] = {
            id: currentUser.id,
            nickname: currentUser.nickname,
            avatar_url: currentUser.avatar_url,
            phone: currentUser.phone,
          }
        }

        if (!cancelled) {
          setSenderProfiles(profiles)
        }
      } catch (profileError) {
        console.error('加载群成员资料失败', profileError)
      }
    }

    void loadSenderProfiles()

    return () => {
      cancelled = true
    }
  }, [
    conversation?.chatType,
    conversation?.id,
    currentUser?.id,
  ])

  useEffect(() => {
    if (!conversation?.id) {
      setMessages([])
      setLoading(false)
      setError('')
      return undefined
    }

    let cancelled = false

    setMessages((current) =>
      current.filter(
        (message) =>
          message.conversation_id === conversation.id,
      ),
    )

    async function loadMessages() {
      setLoading(true)
      setError('')

      let cached = []

      try {
        cached = await loadCachedMessages(
          conversation.id,
        )

        if (!cancelled && cached.length > 0) {
          setMessages((current) =>
            mergeMessages(current, cached),
          )
        }
      } catch (cacheError) {
        console.error(
          '加载本地消息缓存失败',
          cacheError,
        )
      }

      let pending = []

      try {
        pending = await listPendingMessages(
          conversation.id,
        )

        pending = pending.map((message) => ({
          ...message,
          deliveryStatus: 'pending',
        }))

        if (!cancelled && pending.length > 0) {
          setMessages((current) =>
            mergeMessages(current, cached, pending),
          )
        }
      } catch (pendingError) {
        console.error(
          '恢复待发送消息失败',
          pendingError,
        )
      }

      try {
        let cursor = await getSyncCursor(
          conversation.id,
        )

        const cachedMaxSequence = cached.reduce(
          (maximum, message) =>
            Math.max(
              maximum,
              Number(message.sequence || 0),
            ),
          0,
        )

        cursor = Math.max(cursor, cachedMaxSequence)

        if (cursor === 0) {
          cursor = Math.max(
            0,
            Number(conversation.lastSeq || 0) -
            PAGE_SIZE,
          )
        }

        let syncedMessages = mergeMessages(
          pending,
          cached,
        )

        for (let page = 0; page < 10; page += 1) {
          const data = await syncMessages({
            conversationId: conversation.id,
            afterSeq: cursor,
            limit: PAGE_SIZE,
          })

          if (cancelled) {
            return
          }

          const batch = data.list || []

          if (batch.length === 0) {
            break
          }

          syncedMessages = mergeMessages(
            syncedMessages,
            batch,
          )

          const batchMaxSequence = batch.reduce(
            (maximum, message) =>
              Math.max(
                maximum,
                Number(message.sequence || 0),
              ),
            cursor,
          )

          cursor = batchMaxSequence

          await cacheMessages(batch)
          await saveSyncCursor(
            conversation.id,
            cursor,
          )

          if (!data.has_more) {
            break
          }
        }

        if (!cancelled) {
          setMessages((current) =>
            mergeMessages(syncedMessages, current),
          )
        }
      } catch (err) {
        if (!cancelled) {
          setError(err.message || '消息加载失败')
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void loadMessages()

    return () => {
      cancelled = true
    }
  }, [
    conversation?.id,
    conversation?.lastSeq,
    reloadToken,
  ])

  useEffect(() => {
    if (socketStatus === 'connected') {
      return
    }

    setMessages((current) =>
      current.map((message) => {
        if (
          message.sequence > 0 ||
          message.deliveryStatus === 'accepted'
        ) {
          return message
        }

        return {
          ...message,
          deliveryStatus: 'pending',
        }
      }),
    )
  }, [socketStatus])

  useEffect(() => {
    if (!conversation?.id) {
      return undefined
    }

    return chatSocket.subscribe((event) => {
      if (event.type === 'connection.ready') {
        setReloadToken((value) => value + 1)
        return
      }

      if (event.type === 'message.created') {
        const message = event.data

        if (
          !message ||
          message.conversation_id !== conversation.id
        ) {
          return
        }

        void removePendingMessage(
          message.client_message_id,
        )

        void cacheMessages([message])
          .then(() =>
            saveSyncCursor(
              message.conversation_id,
              message.sequence,
            ),
          )
          .catch((error) => {
            console.error('缓存实时消息失败', error)
          })

        setMessages((current) => {
          const withoutSameMessage = current.filter(
            (item) =>
              item.message_id !== message.message_id &&
              item.client_message_id !==
              message.client_message_id,
          )

          return sortMessages([
            ...withoutSameMessage,
            {
              ...message,
              deliveryStatus: 'sent',
            },
          ])
        })

        return
      }

      if (event.type === 'message.rejected') {
        const rejected = event.data

        if (
          !rejected ||
          rejected.conversation_id !== conversation.id
        ) {
          return
        }

        void removePendingMessage(
          rejected.client_message_id,
        )

        setMessages((current) =>
          current.filter(
            (item) =>
              item.client_message_id !==
              rejected.client_message_id,
          ),
        )
        setSendError(
          rejected.message || '消息发送失败',
        )
        return
      }

      if (event.type === 'message.accepted') {
        const accepted = event.data

        if (
          !accepted ||
          accepted.conversation_id !== conversation.id
        ) {
          return
        }

        void updatePendingMessage(
          accepted.client_message_id,
          {
            deliveryStatus: 'accepted',
          },
        )

        setMessages((current) =>
          current.map((item) => {
            if (
              item.client_message_id !==
              accepted.client_message_id
            ) {
              return item
            }

            return {
              ...item,
              deliveryStatus: 'accepted',
            }
          }),
        )
      }

      if (event.type === 'error' && event.error) {
        setSendError(event.error)
      }
    })
  }, [conversation?.id])

  useEffect(() => {
    if (!conversation?.id) {
      return
    }

    const frame = scrollToLatest()

    return () => {
      window.cancelAnimationFrame(frame)
    }
  }, [conversation?.id])

  useEffect(() => {
    const container = scrollContainerRef.current
    const content = scrollContentRef.current

    if (!container || !content) {
      return undefined
    }

    const keepLatestVisible = () => {
      if (suppressAutoScrollRef.current) {
        return
      }

      container.scrollTop = container.scrollHeight
    }

    const observer = new ResizeObserver(
      keepLatestVisible,
    )

    observer.observe(content)
    scrollToLatest()

    return () => {
      observer.disconnect()
    }
  }, [conversation?.id])

  useEffect(() => {
    if (messages.length === 0) {
      return
    }

    const frame = scrollToLatest()

    return () => {
      window.cancelAnimationFrame(frame)
    }
  }, [messages.length])

  useEffect(() => {
    const firstOfficialMessage = messages.find(
      (message) => Number(message.sequence || 0) > 0,
    )

    if (!firstOfficialMessage) {
      setHasOlder(false)
      return
    }

    setHasOlder(firstOfficialMessage.sequence > 1)
  }, [conversation?.id, messages])

  async function handleLoadOlder() {
    if (
      loadingOlder ||
      !conversation?.id ||
      messages.length === 0
    ) {
      return
    }

    const oldestMessage = messages.find(
      (message) => Number(message.sequence || 0) > 0,
    )

    if (!oldestMessage) {
      setHasOlder(false)
      return
    }

    const container = scrollContainerRef.current
    const previousHeight = container?.scrollHeight || 0
    const previousScrollTop = container?.scrollTop || 0

    suppressAutoScrollRef.current = true
    setLoadingOlder(true)
    setOlderError('')

    try {
      const data = await syncMessages({
        conversationId: conversation.id,
        beforeSeq: oldestMessage.sequence,
        limit: PAGE_SIZE,
      })

      const batch = data.list || []

      if (batch.length === 0) {
        setHasOlder(false)
        return
      }

      await cacheMessages(batch)

      setMessages((current) =>
        mergeMessages(batch, current),
      )

      window.requestAnimationFrame(() => {
        window.requestAnimationFrame(() => {
          const currentContainer =
            scrollContainerRef.current

          if (currentContainer) {
            const addedHeight =
              currentContainer.scrollHeight -
              previousHeight

            currentContainer.scrollTop =
              previousScrollTop + addedHeight
          }

          suppressAutoScrollRef.current = false
        })
      })
    } catch (err) {
      suppressAutoScrollRef.current = false
      setOlderError(
        err.message || '更早消息加载失败',
      )
    } finally {
      setLoadingOlder(false)
    }
  }

  async function handleSend() {
    const content = input.trim()

    if (!conversation?.id || !content) {
      return
    }

    const clientMessageID = createID()

    const pendingMessage = {
      message_id: '',
      conversation_id: conversation.id,
      sequence: 0,
      sender_id: currentUser?.id,
      client_message_id: clientMessageID,
      content_type: 'text',
      content,
      created_at: Date.now(),
      deliveryStatus: 'pending',
      retry_count: 0,
    }

    setSendError('')
    setInput('')

    try {
      await queuePendingMessage(pendingMessage)

      setMessages((current) =>
        mergeMessages(current, [pendingMessage]),
      )
    } catch (error) {
      setInput(content)
      setSendError(
        error.message || '消息加入待发送队列失败',
      )
      return
    }

    if (socketStatus !== 'connected') {
      return
    }

    try {
      await updatePendingMessage(clientMessageID, {
        deliveryStatus: 'sending',
        retry_count: 1,
      })

      chatSocket.sendMessage({
        conversationId: conversation.id,
        clientMessageId: clientMessageID,
        content,
      })

      setMessages((current) =>
        current.map((item) => {
          if (
            item.client_message_id !== clientMessageID
          ) {
            return item
          }

          return {
            ...item,
            deliveryStatus: 'sending',
          }
        }),
      )
    } catch {
      await updatePendingMessage(clientMessageID, {
        deliveryStatus: 'pending',
      })
    }
  }

  function handleInputKeyDown(event) {
    if (
      event.key === 'Enter' &&
      !event.shiftKey
    ) {
      event.preventDefault()
      void handleSend()
    }
  }

  if (!conversation) {
    return (
      <Box
        component="main"
        sx={{
          flex: 1,
          display: 'grid',
          placeItems: 'center',
          bgcolor: '#f4f6f9',
        }}
      >
        <Box sx={{ textAlign: 'center' }}>
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
              fontSize: 13,
              color: '#8c95a3',
            }}
          >
            会话消息会显示在这里
          </Typography>
        </Box>
      </Box>
    )
  }

  const connected = socketStatus === 'connected'

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
          bgcolor: 'rgba(255,255,255,0.82)',
          backdropFilter: 'blur(12px)',
        }}
      >
        <Avatar
          src={conversation.avatarUrl || undefined}
          sx={{
            width: 38,
            height: 38,
            mr: 1.5,
            bgcolor: conversation.color,
            fontWeight: 700,
          }}
        >
          {conversation.name.slice(0, 1)}
        </Avatar>

        <Box>
          <Typography
            sx={{
              fontSize: 16,
              fontWeight: 700,
            }}
          >
            {conversation.name}
          </Typography>

          <Typography
            sx={{
              mt: 0.2,
              color: connected
                ? '#2a9d8f'
                : '#9aa3b1',
              fontSize: 11,
            }}
          >
            {socketStatus === 'connecting' &&
              '正在连接'}
            {socketStatus === 'connected' &&
              '实时连接正常'}
            {socketStatus === 'disconnected' &&
              '连接已断开，正在重连'}
            {socketStatus === 'closed' &&
              '实时连接已关闭'}
          </Typography>
        </Box>

        <Box sx={{ flex: 1 }} />

        <Tooltip title="刷新消息">
          <span>
            <IconButton
              onClick={() => {
                setReloadToken((value) => value + 1)
              }}
              disabled={loading}
            >
              <RefreshRoundedIcon />
            </IconButton>
          </span>
        </Tooltip>

        <IconButton size="small">
          <MoreHorizRoundedIcon />
        </IconButton>
      </Box>

      <Box
        ref={scrollContainerRef}
        className="chat-scroll"
        sx={{
          flex: 1,
          minHeight: 0,
          overflowY: 'auto',
          overflowAnchor: 'none',
          overscrollBehavior: 'contain',
          px: { xs: 2, md: 4 },
          py: 3,
        }}
      >
        <Box
          ref={scrollContentRef}
          sx={{
            width: '100%',
            maxWidth: 860,
            mx: 'auto',
          }}
        >
          {hasOlder && (
            <Box
              sx={{
                mb: 2,
                textAlign: 'center',
              }}
            >
              <Button
                size="small"
                variant="outlined"
                disabled={loadingOlder}
                onClick={() => {
                  void handleLoadOlder()
                }}
              >
                {loadingOlder
                  ? '加载中...'
                  : '加载更早消息'}
              </Button>
            </Box>
          )}

          {olderError && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {olderError}
            </Alert>
          )}
          {error && (
            <Alert
              severity="error"
              action={
                <Button
                  color="inherit"
                  size="small"
                  onClick={() => {
                    setReloadToken((value) => value + 1)
                  }}
                >
                  重试
                </Button>
              }
              sx={{ mb: 2 }}
            >
              {error}
            </Alert>
          )}

          {sendError && (
            <Alert
              severity="error"
              onClose={() => setSendError('')}
              sx={{ mb: 2 }}
            >
              {sendError}
            </Alert>
          )}

          {loading && messages.length === 0 && (
            <Box
              sx={{
                minHeight: 280,
                display: 'grid',
                placeItems: 'center',
              }}
            >
              <CircularProgress size={30} />
            </Box>
          )}

          {!loading &&
            messages.length === 0 &&
            !error && (
              <Box
                sx={{
                  py: 12,
                  textAlign: 'center',
                }}
              >
                <Typography
                  sx={{
                    fontSize: 15,
                    fontWeight: 600,
                  }}
                >
                  还没有消息
                </Typography>

                <Typography
                  sx={{
                    mt: 0.8,
                    fontSize: 13,
                    color: '#8c95a3',
                  }}
                >
                  输入一条消息开始聊天
                </Typography>
              </Box>
            )}

          {messages.map((message) => {
            const own =
              message.sender_id === currentUser?.id
            const senderProfile =
              senderProfiles[message.sender_id]
            const senderName =
              own
                ? currentUser?.nickname ||
                currentUser?.id
                : senderProfile?.nickname ||
                message.sender_id

            return (
              <Box
                key={
                  message.message_id ||
                  message.client_message_id
                }
                sx={{
                  display: 'flex',
                  justifyContent: own
                    ? 'flex-end'
                    : 'flex-start',
                  mb: 1.8,
                }}
              >
                {!own && (
                  <Avatar
                    src={
                      senderProfile?.avatar_url ||
                      undefined
                    }
                    sx={{
                      width: 34,
                      height: 34,
                      mr: 1,
                      bgcolor: '#6d7cff',
                      fontSize: 13,
                    }}
                  >
                    {senderName.slice(0, 1)}
                  </Avatar>
                )}

                <Box
                  sx={{
                    maxWidth: {
                      xs: '78%',
                      md: '64%',
                    },
                  }}
                >
                  {!own &&
                    conversation.chatType === 1 && (
                      <Typography
                        sx={{
                          mb: 0.4,
                          color: '#7f8998',
                          fontSize: 10,
                        }}
                      >
                        {senderName}
                      </Typography>
                    )}

                  <Paper
                    elevation={0}
                    sx={{
                      px: 1.6,
                      py: 1.1,
                      bgcolor: own
                        ? '#2f6fed'
                        : '#ffffff',
                      color: own
                        ? '#ffffff'
                        : '#273142',
                      border: own
                        ? 'none'
                        : '1px solid #e3e7ed',
                      borderRadius: own
                        ? '14px 14px 4px 14px'
                        : '14px 14px 14px 4px',
                      opacity:
                        message.deliveryStatus ===
                          'sending'
                          ? 0.7
                          : 1,
                    }}
                  >
                    <Typography
                      sx={{
                        fontSize: 14,
                        lineHeight: 1.65,
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-word',
                      }}
                    >
                      {message.content}
                    </Typography>
                  </Paper>

                  <Typography
                    sx={{
                      mt: 0.45,
                      color: '#9aa3b1',
                      fontSize: 10,
                      textAlign: own
                        ? 'right'
                        : 'left',
                    }}
                  >
                    {formatMessageTime(
                      message.created_at,
                    )}

                    {own &&
                      message.deliveryStatus === 'pending' &&
                      ' · 等待发送'}

                    {own &&
                      message.deliveryStatus === 'sending' &&
                      ' · 发送中'}

                    {own &&
                      message.deliveryStatus === 'accepted' &&
                      ' · 已提交'}
                  </Typography>
                </Box>
              </Box>
            )
          })}
        </Box>
      </Box>

      <Divider />

      <Box
        sx={{
          p: 2,
          bgcolor: '#f4f6f9',
        }}
      >
        <Box
          sx={{
            display: 'flex',
            alignItems: 'flex-end',
            gap: 1,
          }}
        >
          <TextField
            value={input}
            onChange={(event) => {
              setInput(event.target.value)
            }}
            onKeyDown={handleInputKeyDown}
            multiline
            minRows={2}
            maxRows={4}
            placeholder={
              connected
                ? '输入消息，Enter 发送，Shift+Enter 换行'
                : '当前离线，发送后会进入待发送队列'
            }
            fullWidth
            sx={{
              '& .MuiOutlinedInput-root': {
                bgcolor: '#ffffff',
              },
            }}
          />

          <Button
            variant="contained"
            disabled={!input.trim()}
            onClick={handleSend}
            sx={{
              minWidth: 48,
              width: 48,
              height: 48,
              p: 0,
            }}
          >
            <SendRoundedIcon />
          </Button>
        </Box>
      </Box>
    </Box>
  )
}

function mergeMessages(...groups) {
  const result = new Map()

  for (const group of groups) {
    for (const message of group || []) {
      const key = message.client_message_id
        ? `client:${message.client_message_id}`
        : `message:${message.message_id}`

      const existing = result.get(key)

      if (existing?.message_id && !message.message_id) {
        continue
      }

      result.set(key, message)
    }
  }

  return sortMessages([...result.values()])
}
