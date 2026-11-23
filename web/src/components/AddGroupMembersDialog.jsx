import { useEffect, useMemo, useState } from 'react'
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

import { addGroupMembers } from '../api/social'

export default function AddGroupMembersDialog({
  open,
  groupId,
  friends,
  existingMemberIDs,
  onClose,
  onAdded,
}) {
  const [selectedFriends, setSelectedFriends] = useState([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const options = useMemo(() => {
    const existing = new Set(existingMemberIDs)

    return friends.filter(
      (friend) => !existing.has(friend.id),
    )
  }, [existingMemberIDs, friends])

  useEffect(() => {
    if (!open) {
      return
    }

    setSelectedFriends([])
    setError('')
    setLoading(false)
  }, [open])

  async function handleSubmit(event) {
    event.preventDefault()

    if (selectedFriends.length === 0) {
      setError('请选择要添加的好友')
      return
    }

    setError('')
    setLoading(true)

    try {
      const data = await addGroupMembers(
        groupId,
        selectedFriends.map((friend) => friend.id),
      )

      await onAdded(data.added_user_ids || [])
      onClose()
    } catch (err) {
      setError(err.message || '添加群成员失败')
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
          添加群成员
        </DialogTitle>

        <DialogContent>
          <Stack spacing={2.2} sx={{ pt: 1 }}>
            {error && (
              <Alert severity="error">
                {error}
              </Alert>
            )}

            <Autocomplete
              multiple
              options={options}
              value={selectedFriends}
              onChange={(_, value) => {
                setSelectedFriends(value)
              }}
              getOptionLabel={(option) =>
                option.name || option.id
              }
              isOptionEqualToValue={(option, value) =>
                option.id === value.id
              }
              noOptionsText="没有可添加的好友"
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
                    {(option.name || option.id).slice(0, 1)}
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
                  label="选择好友"
                  placeholder="可以选择多个好友"
                  autoFocus
                />
              )}
            />
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
            disabled={loading || selectedFriends.length === 0}
          >
            {loading ? '添加中...' : '添加成员'}
          </Button>
        </DialogActions>
      </Box>
    </Dialog>
  )
}