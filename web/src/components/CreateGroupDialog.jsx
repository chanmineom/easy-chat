import { useEffect, useState } from 'react'
import {
  Alert,
  Autocomplete,
  Avatar,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Stack,
  TextField,
  Typography,
} from '@mui/material'

import { createGroup } from '../api/social'

export default function CreateGroupDialog({
  open,
  friends,
  onClose,
  onCreated,
}) {
  const [name, setName] = useState('')
  const [notification, setNotification] = useState('')
  const [selectedFriends, setSelectedFriends] = useState([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!open) {
      return
    }

    setName('')
    setNotification('')
    setSelectedFriends([])
    setError('')
    setLoading(false)
  }, [open])

  async function handleSubmit(event) {
    event.preventDefault()

    const groupName = name.trim()

    if (!groupName) {
      setError('请输入群名称')
      return
    }

    setError('')
    setLoading(true)

    try {
      const data = await createGroup({
        name: groupName,
        notification: notification.trim(),
        memberIds: selectedFriends.map(
          (friend) => friend.id,
        ),
      })

      await onCreated(data.group_id)
      onClose()
    } catch (err) {
      setError(err.message || '群组创建失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog
      open={open}
      onClose={loading ? undefined : onClose}
      fullWidth
      maxWidth="sm"
    >
      <Box component="form" onSubmit={handleSubmit}>
        <DialogTitle sx={{ fontWeight: 700 }}>
          创建群组
        </DialogTitle>

        <DialogContent>
          <Stack spacing={2.2} sx={{ pt: 1 }}>
            {error && (
              <Alert severity="error">
                {error}
              </Alert>
            )}

            <TextField
              label="群名称"
              value={name}
              onChange={(event) => {
                setName(event.target.value)
              }}
              autoFocus
              fullWidth
            />

            <TextField
              label="群公告"
              value={notification}
              onChange={(event) => {
                setNotification(event.target.value)
              }}
              multiline
              minRows={3}
              fullWidth
            />

            <Autocomplete
              multiple
              options={friends}
              value={selectedFriends}
              onChange={(_, value) => {
                setSelectedFriends(value)
              }}
              getOptionLabel={(option) =>
                option.name || option.id
              }
              isOptionEqualToValue={(
                option,
                value,
              ) => option.id === value.id}
              renderOption={(props, option) => (
                <Box
                  component="li"
                  {...props}
                  sx={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 1,
                  }}
                >
                  <Avatar
                    src={option.avatarUrl || undefined}
                    sx={{
                      width: 30,
                      height: 30,
                      bgcolor: '#2f6fed',
                      fontSize: 12,
                    }}
                  >
                    {(option.name || option.id)
                      .slice(0, 1)}
                  </Avatar>

                  <Box>
                    <Typography sx={{ fontSize: 13 }}>
                      {option.name || option.id}
                    </Typography>

                    <Typography
                      sx={{
                        color: '#8c95a3',
                        fontSize: 11,
                      }}
                    >
                      {option.phone || option.id}
                    </Typography>
                  </Box>
                </Box>
              )}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="选择群成员"
                  placeholder="从好友中选择"
                />
              )}
            />

            <Typography
              sx={{
                color: '#8c95a3',
                fontSize: 12,
              }}
            >
              你自己会自动成为群成员，不需要重复选择。
            </Typography>
          </Stack>
        </DialogContent>

        <DialogActions sx={{ px: 3, pb: 2.5 }}>
          <Button
            onClick={onClose}
            disabled={loading}
          >
            取消
          </Button>

          <Button
            type="submit"
            variant="contained"
            disabled={loading || !name.trim()}
          >
            {loading ? '创建中...' : '创建群组'}
          </Button>
        </DialogActions>
      </Box>
    </Dialog>
  )
}