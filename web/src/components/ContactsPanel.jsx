import {
  useCallback,
  useEffect,
  useState,
} from 'react'
import {
  Alert,
  Avatar,
  Box,
  Button,
  Chip,
  CircularProgress,
  IconButton,
  List,
  ListItem,
  ListItemAvatar,
  ListItemText,
  Paper,
  Snackbar,
  Tooltip,
  Typography,
} from '@mui/material'

import AddRoundedIcon from '@mui/icons-material/AddRounded'
import CheckRoundedIcon from '@mui/icons-material/CheckRounded'
import CloseRoundedIcon from '@mui/icons-material/CloseRounded'
import ContentCopyRoundedIcon from '@mui/icons-material/ContentCopyRounded'
import DeleteOutlineRoundedIcon from '@mui/icons-material/DeleteOutlineRounded'
import RefreshRoundedIcon from '@mui/icons-material/RefreshRounded'

import {
  handleFriendRequest,
  listFriends,
  listIncomingFriendRequests,
  removeFriend,
} from '../api/social'
import { getUsers } from '../api/users'
import { useAuthStore } from '../stores/authStore'
import AddFriendDialog from './AddFriendDialog'

const initialData = {
  friendIDs: [],
  requests: [],
  profiles: {},
}

function getDisplayName(profile, userID) {
  return profile?.nickname || userID
}

function getAvatarLetter(profile, userID) {
  return (
    profile?.nickname?.slice(0, 1) ||
    userID.slice(-1).toUpperCase()
  )
}

function maskPhone(phone) {
  if (!phone) {
    return ''
  }

  return phone.replace(
    /^(\d{3})\d+(\d{4})$/,
    '$1****$2',
  )
}

function formatTime(timestamp) {
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

function fallbackCopyText(value) {
  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.setAttribute('readonly', '')
  textarea.style.position = 'fixed'
  textarea.style.top = '-9999px'
  textarea.style.opacity = '0'

  document.body.appendChild(textarea)
  textarea.focus()
  textarea.select()
  textarea.setSelectionRange(0, value.length)

  let copied = false

  try {
    copied = document.execCommand('copy')
  } finally {
    document.body.removeChild(textarea)
  }

  return copied
}

export default function ContactsPanel() {
  const user = useAuthStore((state) => state.user)

  const [data, setData] = useState(initialData)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [profileWarning, setProfileWarning] = useState('')
  const [actionKey, setActionKey] = useState('')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [notice, setNotice] = useState({
    open: false,
    message: '',
    severity: 'success',
  })

  const loadContacts = useCallback(async ({
    silent = false,
  } = {}) => {
    if (!silent) {
      setLoading(true)
    }

    setError('')
    setProfileWarning('')

    try {
      const [
        friendResult,
        requestResult,
      ] = await Promise.all([
        listFriends(),
        listIncomingFriendRequests(),
      ])

      const friendIDs = friendResult.friend_ids || []
      const requests = (
        requestResult.list || []
      ).filter((item) => item.status === 1)

      const profileIDs = [
        ...friendIDs,
        ...requests.map((item) => item.requester_id),
      ]

      let profiles = {}

      try {
        const users = await getUsers(profileIDs)

        profiles = Object.fromEntries(
          users.map((item) => [item.id, item]),
        )
      } catch (profileErr) {
        setProfileWarning(
          profileErr.message ||
            '用户资料加载失败，当前显示用户 ID',
        )
      }

      setData({
        friendIDs,
        requests,
        profiles,
      })
    } catch (loadErr) {
      setError(loadErr.message || '联系人加载失败')
    } finally {
      if (!silent) {
        setLoading(false)
      }
    }
  }, [])

  useEffect(() => {
    void loadContacts()

    const timer = window.setInterval(() => {
      void loadContacts({ silent: true })
    }, 10000)

    return () => {
      window.clearInterval(timer)
    }
  }, [loadContacts])

  function showNotice(message, severity = 'success') {
    setNotice({
      open: true,
      message,
      severity,
    })
  }

  async function copyUserID() {
    if (!user?.id) {
      return
    }

    try {
      await navigator.clipboard.writeText(user.id)
      showNotice('用户 ID 已复制')
    } catch {
      if (fallbackCopyText(user.id)) {
        showNotice('用户 ID 已复制')
        return
      }

      showNotice('复制失败，请手动选择用户 ID', 'error')
    }
  }

  async function handleRequest(requestID, accept) {
    const key = `${accept ? 'accept' : 'reject'}-${requestID}`
    setActionKey(key)

    try {
      await handleFriendRequest({
        requestId: requestID,
        accept,
      })

      await loadContacts()
      showNotice(accept ? '已添加为好友' : '已拒绝申请')
    } catch (err) {
      showNotice(
        err.message || '处理好友申请失败',
        'error',
      )
    } finally {
      setActionKey('')
    }
  }

  async function handleRemove(friendID) {
    if (!window.confirm('确定删除这个好友吗？')) {
      return
    }

    const key = `remove-${friendID}`
    setActionKey(key)

    try {
      await removeFriend(friendID)
      await loadContacts()
      showNotice('好友已删除')
    } catch (err) {
      showNotice(
        err.message || '删除好友失败',
        'error',
      )
    } finally {
      setActionKey('')
    }
  }

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
        <Box>
          <Typography
            sx={{
              fontSize: 20,
              fontWeight: 750,
              letterSpacing: '-0.3px',
            }}
          >
            联系人
          </Typography>

          <Typography
            sx={{
              mt: 0.2,
              fontSize: 11,
              color: '#8c95a3',
            }}
          >
            {data.friendIDs.length} 位好友
          </Typography>
        </Box>

        <Box sx={{ flex: 1 }} />

        <Tooltip title="刷新">
          <span>
            <IconButton
              onClick={() => {
                void loadContacts()
              }}
              disabled={loading}
            >
              <RefreshRoundedIcon />
            </IconButton>
          </span>
        </Tooltip>

        <Button
          variant="contained"
          startIcon={<AddRoundedIcon />}
          onClick={() => setDialogOpen(true)}
          sx={{ ml: 1 }}
        >
          添加好友
        </Button>
      </Box>

      <Box
        className="chat-scroll"
        sx={{
          flex: 1,
          overflowY: 'auto',
          px: { xs: 2, md: 4 },
          py: 3,
        }}
      >
        <Box
          sx={{
            width: '100%',
            maxWidth: 900,
            mx: 'auto',
          }}
        >
          <Paper
            elevation={0}
            sx={{
              mb: 3,
              p: 2.5,
              display: 'flex',
              alignItems: 'center',
              gap: 2,
              border: '1px solid #e5e8ed',
            }}
          >
            <Avatar
              src={user?.avatar_url || undefined}
              sx={{
                width: 48,
                height: 48,
                bgcolor: '#2f6fed',
                fontWeight: 700,
              }}
            >
              {user?.nickname?.slice(0, 1) || 'U'}
            </Avatar>

            <Box sx={{ minWidth: 0, flex: 1 }}>
              <Typography
                sx={{
                  fontSize: 14,
                  fontWeight: 700,
                }}
              >
                {user?.nickname || '我的账号'}
              </Typography>

              <Typography
                sx={{
                  mt: 0.3,
                  fontSize: 11,
                  color: '#8c95a3',
                }}
              >
                我的用户 ID
              </Typography>

              <Typography
                component="code"
                sx={{
                  display: 'block',
                  mt: 0.4,
                  overflow: 'hidden',
                  color: '#485466',
                  fontFamily: 'monospace',
                  fontSize: 12,
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                  userSelect: 'all',
                }}
              >
                {user?.id || '加载中'}
              </Typography>
            </Box>

            <Tooltip title="复制用户 ID">
              <span>
                <IconButton
                  onClick={copyUserID}
                  disabled={!user?.id}
                >
                  <ContentCopyRoundedIcon />
                </IconButton>
              </span>
            </Tooltip>
          </Paper>

          {error && (
            <Alert
              severity="error"
              action={
                <Button
                  color="inherit"
                  size="small"
                  onClick={() => {
                    void loadContacts()
                  }}
                >
                  重试
                </Button>
              }
              sx={{ mb: 3 }}
            >
              {error}
            </Alert>
          )}

          {profileWarning && (
            <Alert severity="warning" sx={{ mb: 3 }}>
              {profileWarning}
            </Alert>
          )}

          {loading &&
          data.friendIDs.length === 0 &&
          data.requests.length === 0 ? (
            <Box
              sx={{
                minHeight: 300,
                display: 'grid',
                placeItems: 'center',
              }}
            >
              <CircularProgress size={30} />
            </Box>
          ) : (
            <>
              <Typography
                sx={{
                  mb: 1.5,
                  fontSize: 14,
                  fontWeight: 700,
                }}
              >
                新的朋友
                {data.requests.length > 0 && (
                  <Chip
                    size="small"
                    color="primary"
                    label={data.requests.length}
                    sx={{ ml: 1, height: 20 }}
                  />
                )}
              </Typography>

              <Paper
                elevation={0}
                sx={{
                  mb: 3,
                  border: '1px solid #e5e8ed',
                  overflow: 'hidden',
                }}
              >
                {data.requests.length === 0 ? (
                  <Typography
                    sx={{
                      px: 2.5,
                      py: 3,
                      fontSize: 13,
                      color: '#8c95a3',
                    }}
                  >
                    暂无新的好友申请
                  </Typography>
                ) : (
                  <List disablePadding>
                    {data.requests.map((request) => {
                      const profile =
                        data.profiles[request.requester_id]
                      const acceptKey =
                        `accept-${request.id}`
                      const rejectKey =
                        `reject-${request.id}`

                      return (
                        <ListItem
                          key={request.id}
                          divider
                          sx={{ py: 1.4 }}
                          secondaryAction={
                            <Box
                              sx={{
                                display: 'flex',
                                gap: 0.7,
                              }}
                            >
                              <Button
                                size="small"
                                variant="contained"
                                startIcon={<CheckRoundedIcon />}
                                disabled={Boolean(actionKey)}
                                onClick={() => {
                                  void handleRequest(
                                    request.id,
                                    true,
                                  )
                                }}
                              >
                                {actionKey === acceptKey
                                  ? '处理中'
                                  : '接受'}
                              </Button>

                              <Button
                                size="small"
                                color="inherit"
                                startIcon={<CloseRoundedIcon />}
                                disabled={Boolean(actionKey)}
                                onClick={() => {
                                  void handleRequest(
                                    request.id,
                                    false,
                                  )
                                }}
                              >
                                {actionKey === rejectKey
                                  ? '处理中'
                                  : '拒绝'}
                              </Button>
                            </Box>
                          }
                        >
                          <ListItemAvatar>
                            <Avatar
                              src={
                                profile?.avatar_url ||
                                undefined
                              }
                              sx={{ bgcolor: '#6d7cff' }}
                            >
                              {getAvatarLetter(
                                profile,
                                request.requester_id,
                              )}
                            </Avatar>
                          </ListItemAvatar>

                          <ListItemText
                            primary={getDisplayName(
                              profile,
                              request.requester_id,
                            )}
                            secondary={
                              <>
                                <Typography
                                  component="span"
                                  sx={{
                                    display: 'block',
                                    fontSize: 12,
                                    color: '#7d8796',
                                  }}
                                >
                                  {request.message ||
                                    '请求添加你为好友'}
                                </Typography>
                                <Typography
                                  component="span"
                                  sx={{
                                    display: 'block',
                                    mt: 0.3,
                                    color: '#a0a8b4',
                                    fontFamily: 'monospace',
                                    fontSize: 11,
                                    userSelect: 'all',
                                  }}
                                >
                                  ID: {request.requester_id}
                                </Typography>
                                <Typography
                                  component="span"
                                  sx={{
                                    display: 'block',
                                    mt: 0.3,
                                    fontSize: 11,
                                    color: '#a0a8b4',
                                  }}
                                >
                                  {formatTime(
                                    request.created_at,
                                  )}
                                </Typography>
                              </>
                            }
                            sx={{ pr: 22 }}
                          />
                        </ListItem>
                      )
                    })}
                  </List>
                )}
              </Paper>

              <Box
                sx={{
                  mb: 1.5,
                  display: 'flex',
                  alignItems: 'center',
                }}
              >
                <Typography
                  sx={{
                    fontSize: 14,
                    fontWeight: 700,
                  }}
                >
                  好友
                </Typography>

                <Typography
                  sx={{
                    ml: 1,
                    fontSize: 12,
                    color: '#99a1ae',
                  }}
                >
                  {data.friendIDs.length}
                </Typography>
              </Box>

              <Paper
                elevation={0}
                sx={{
                  border: '1px solid #e5e8ed',
                  overflow: 'hidden',
                }}
              >
                {data.friendIDs.length === 0 ? (
                  <Box
                    sx={{
                      py: 8,
                      textAlign: 'center',
                    }}
                  >
                    <Typography
                      sx={{
                        fontSize: 15,
                        fontWeight: 600,
                      }}
                    >
                      还没有好友
                    </Typography>
                    <Typography
                      sx={{
                        mt: 0.8,
                        fontSize: 13,
                        color: '#8c95a3',
                      }}
                    >
                      点击右上角添加好友
                    </Typography>
                  </Box>
                ) : (
                  <List disablePadding>
                    {data.friendIDs.map((friendID) => {
                      const profile = data.profiles[friendID]
                      const removeKey =
                        `remove-${friendID}`

                      return (
                        <ListItem
                          key={friendID}
                          divider
                          sx={{ py: 1.4 }}
                          secondaryAction={
                            <Tooltip title="删除好友">
                              <span>
                                <IconButton
                                  edge="end"
                                  color="error"
                                  disabled={Boolean(actionKey)}
                                  onClick={() => {
                                    void handleRemove(friendID)
                                  }}
                                >
                                  {actionKey === removeKey ? (
                                    <CircularProgress size={18} />
                                  ) : (
                                    <DeleteOutlineRoundedIcon />
                                  )}
                                </IconButton>
                              </span>
                            </Tooltip>
                          }
                        >
                          <ListItemAvatar>
                            <Avatar
                              src={
                                profile?.avatar_url ||
                                undefined
                              }
                              sx={{ bgcolor: '#2f6fed' }}
                            >
                              {getAvatarLetter(
                                profile,
                                friendID,
                              )}
                            </Avatar>
                          </ListItemAvatar>

                          <ListItemText
                            primary={getDisplayName(
                              profile,
                              friendID,
                            )}
                            secondary={
                              maskPhone(profile?.phone) ||
                              friendID
                            }
                            primaryTypographyProps={{
                              fontSize: 14,
                              fontWeight: 600,
                            }}
                            secondaryTypographyProps={{
                              fontSize: 12,
                            }}
                          />
                        </ListItem>
                      )
                    })}
                  </List>
                )}
              </Paper>
            </>
          )}
        </Box>
      </Box>

      <AddFriendDialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
        onSent={async () => {
          await loadContacts()
          showNotice('好友申请已发送')
        }}
      />

      <Snackbar
        open={notice.open}
        autoHideDuration={2500}
        onClose={() => {
          setNotice((current) => ({
            ...current,
            open: false,
          }))
        }}
        anchorOrigin={{
          vertical: 'bottom',
          horizontal: 'center',
        }}
      >
        <Alert
          severity={notice.severity}
          variant="filled"
          onClose={() => {
            setNotice((current) => ({
              ...current,
              open: false,
            }))
          }}
        >
          {notice.message}
        </Alert>
      </Snackbar>
    </Box>
  )
}
