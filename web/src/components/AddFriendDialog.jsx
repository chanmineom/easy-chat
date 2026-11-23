import { useEffect, useState } from 'react'
import {
    Alert,
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

import { sendFriendRequest } from '../api/social'

export default function AddFriendDialog({
    open,
    onClose,
    onSent,
}) {
    const [recipientID, setRecipientID] = useState('')
    const [message, setMessage] = useState(
        '你好，我想加你为好友',
    )
    const [error, setError] = useState('')
    const [loading, setLoading] = useState(false)

    useEffect(() => {
        if (!open) {
            return
        }

        setRecipientID('')
        setMessage('你好，我想加你为好友')
        setError('')
        setLoading(false)
    }, [open])

    async function handleSubmit(event) {
        event.preventDefault()

        const trimmedID = recipientID.trim()
        if (!trimmedID) {
            setError('请输入对方用户 ID')
            return
        }

        setError('')
        setLoading(true)

        try {
            await sendFriendRequest({
                recipientId: trimmedID,
                message: message.trim(),
            })

            await onSent()
            onClose()
        } catch (err) {
            setError(err.message || '好友申请发送失败')
        } finally {
            setLoading(false)
        }
    }

    return (
        <Dialog
            open={open}
            onClose={loading ? undefined : onClose}
            fullWidth
            maxWidth="xs"
        >
            <Box onSubmit={handleSubmit} component="form">
                <DialogTitle sx={{ fontWeight: 700 }}>
                    添加好友
                </DialogTitle>

                <DialogContent>
                    <Stack spacing={2.2} sx={{ pt: 1 }}>
                        <Typography
                            sx={{
                                fontSize: 13,
                                color: '#7b8494',
                            }}
                        >
                            当前先通过用户 ID 添加，后续会增加手机号搜索。
                        </Typography>

                        {error && (
                            <Alert severity="error">
                                {error}
                            </Alert>
                        )}

                        <TextField
                            label="对方用户 ID"
                            value={recipientID}
                            onChange={(event) => {
                                setRecipientID(event.target.value)
                            }}
                            placeholder="u_xxxxxxxxx"
                            autoFocus
                            fullWidth
                        />

                        <TextField
                            label="验证消息"
                            value={message}
                            onChange={(event) => {
                                setMessage(event.target.value)
                            }}
                            multiline
                            minRows={2}
                            inputProps={{ maxLength: 255 }}
                            fullWidth
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
                        disabled={loading || !recipientID.trim()}
                    >
                        {loading ? '发送中...' : '发送申请'}
                    </Button>
                </DialogActions>
            </Box>
        </Dialog>
    )
}