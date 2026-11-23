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
    Tooltip,
    List,
    ListItem,
    ListItemAvatar,
    ListItemButton,
    ListItemText,
    Paper,
    Snackbar,
    Typography,
    Menu,
    MenuItem,
} from '@mui/material'

import AddRoundedIcon from '@mui/icons-material/AddRounded'
import GroupsRoundedIcon from '@mui/icons-material/GroupsRounded'
import RefreshRoundedIcon from '@mui/icons-material/RefreshRounded'
import DeleteOutlineRoundedIcon from '@mui/icons-material/DeleteOutlineRounded'
import LogoutRoundedIcon from '@mui/icons-material/LogoutRounded'
import PersonAddAltRoundedIcon from '@mui/icons-material/PersonAddAltRounded'
import MoreVertRoundedIcon from '@mui/icons-material/MoreVertRounded'

import {
    getGroup,
    kickGroupMember,
    leaveGroup,
    listFriends,
    listGroupMembers,
    listGroups,
    changeGroupMemberRole,
    dissolveGroup,
    transferGroupOwnership,
} from '../api/social'
import { getUsers } from '../api/users'
import { useAuthStore } from '../stores/authStore'
import CreateGroupDialog from './CreateGroupDialog'
import AddGroupMembersDialog from './AddGroupMembersDialog'

function roleLabel(role) {
    switch (role) {
        case 1:
            return '群主'
        case 2:
            return '管理员'
        case 3:
            return '成员'
        default:
            return '成员'
    }
}

function colorFor(value) {
    const colors = [
        '#2f6fed',
        '#2a9d8f',
        '#ef8354',
        '#8b5cf6',
        '#d65a7a',
    ]

    let total = 0

    for (const char of value || '') {
        total += char.charCodeAt(0)
    }

    return colors[total % colors.length]
}

export default function GroupsPanel() {
    const currentUser = useAuthStore((state) => state.user)

    const [groups, setGroups] = useState([])
    const [selectedID, setSelectedID] = useState('')
    const [groupDetail, setGroupDetail] = useState(null)
    const [members, setMembers] = useState([])
    const [profiles, setProfiles] = useState({})
    const [friends, setFriends] = useState([])

    const [loadingGroups, setLoadingGroups] = useState(true)
    const [loadingDetail, setLoadingDetail] = useState(false)
    const [error, setError] = useState('')
    const [detailError, setDetailError] = useState('')
    const [dialogOpen, setDialogOpen] = useState(false)
    const [addMembersOpen, setAddMembersOpen] = useState(false)
    const [detailReloadToken, setDetailReloadToken] = useState(0)
    const [actionKey, setActionKey] = useState('')

    const [memberMenuAnchor, setMemberMenuAnchor] = useState(null)
    const [memberMenuTarget, setMemberMenuTarget] = useState(null)

    const [notice, setNotice] = useState({
        open: false,
        message: '',
        severity: 'success',
    })

    const loadGroups = useCallback(async () => {
        setLoadingGroups(true)
        setError('')

        try {
            const data = await listGroups()
            const list = data.list || []

            setGroups(list)
            setSelectedID((current) => {
                if (
                    current &&
                    list.some(
                        (group) => group.group_id === current,
                    )
                ) {
                    return current
                }

                return list[0]?.group_id || ''
            })
        } catch (err) {
            setError(err.message || '群组加载失败')
        } finally {
            setLoadingGroups(false)
        }
    }, [])

    const loadFriendOptions = useCallback(async () => {
        try {
            const friendData = await listFriends()
            const friendIDs = friendData.friend_ids || []
            const users = await getUsers(friendIDs)

            setFriends(
                users.map((user) => ({
                    id: user.id,
                    name: user.nickname,
                    phone: user.phone,
                    avatarUrl: user.avatar_url,
                })),
            )
        } catch (err) {
            console.error('加载好友选项失败', err)
        }
    }, [])

    useEffect(() => {
        void loadGroups()
        void loadFriendOptions()
    }, [loadFriendOptions, loadGroups])

    useEffect(() => {
        if (!selectedID) {
            setGroupDetail(null)
            setMembers([])
            setProfiles({})
            return undefined
        }

        let cancelled = false

        async function loadDetail() {
            setLoadingDetail(true)
            setDetailError('')

            try {
                const [groupData, memberData] =
                    await Promise.all([
                        getGroup(selectedID),
                        listGroupMembers(selectedID),
                    ])

                if (cancelled) {
                    return
                }

                const memberList = memberData.list || []
                const memberIDs = memberList.map(
                    (member) => member.user_id,
                )
                const users = await getUsers(memberIDs)

                if (cancelled) {
                    return
                }

                setGroupDetail(groupData.group)
                setMembers(memberList)
                setProfiles(
                    Object.fromEntries(
                        users.map((user) => [user.id, user]),
                    ),
                )
            } catch (err) {
                if (!cancelled) {
                    setDetailError(
                        err.message || '群组详情加载失败',
                    )
                }
            } finally {
                if (!cancelled) {
                    setLoadingDetail(false)
                }
            }
        }

        void loadDetail()

        return () => {
            cancelled = true
        }
    }, [selectedID, detailReloadToken])

    function showNotice(message, severity = 'success') {
        setNotice({
            open: true,
            message,
            severity,
        })
    }

    async function handleCreated(groupID) {
        await loadGroups()
        setSelectedID(groupID)
        showNotice('群组创建成功')
    }

    function canManageGroup() {
        return (
            groupDetail?.role === 1 ||
            groupDetail?.role === 2
        )
    }

    function canKickMember(member) {
        if (!groupDetail) {
            return false
        }

        if (member.user_id === currentUser?.id) {
            return false
        }

        if (member.role === 1) {
            return false
        }

        if (groupDetail.role === 1) {
            return true
        }

        if (groupDetail.role === 2) {
            return member.role === 3
        }

        return false
    }

    function canManageMemberRole(member) {
        if (!groupDetail || groupDetail.role !== 1) {
            return false
        }

        if (!member) {
            return false
        }

        if (member.user_id === currentUser?.id) {
            return false
        }

        return member.role !== 1
    }

    function isOwner() {
        return groupDetail?.role === 1
    }

    async function handleKickMember(member) {
        const profile = profiles[member.user_id]
        const name = profile?.nickname || member.user_id

        if (!window.confirm(`确定将「${name}」移出群组吗？`)) {
            return
        }

        const key = `kick-${member.user_id}`
        setActionKey(key)

        try {
            await kickGroupMember(
                selectedID,
                member.user_id,
            )

            await loadGroups()
            setDetailReloadToken((value) => value + 1)
            showNotice('成员已移出群组')
        } catch (err) {
            showNotice(
                err.message || '移除成员失败',
                'error',
            )
        } finally {
            setActionKey('')
        }
    }

    async function handleLeaveGroup() {
        if (!groupDetail) {
            return
        }

        if (
            !window.confirm(
                `确定退出群组「${groupDetail.name}」吗？`,
            )
        ) {
            return
        }

        setActionKey('leave')

        try {
            await leaveGroup(selectedID)
            await loadGroups()
            showNotice('已退出群组')
        } catch (err) {
            showNotice(
                err.message || '退出群组失败',
                'error',
            )
        } finally {
            setActionKey('')
        }
    }

    async function handleChangeRole(member, role) {
        const profile = profiles[member.user_id]
        const name = profile?.nickname || member.user_id

        const actionText =
            role === 2 ? '设为管理员' : '取消管理员'

        if (
            !window.confirm(
                `确定将「${name}」${actionText}吗？`,
            )
        ) {
            return
        }

        const key = `role-${member.user_id}-${role}`
        setActionKey(key)
        setMemberMenuAnchor(null)

        try {
            await changeGroupMemberRole({
                groupId: selectedID,
                userId: member.user_id,
                role,
            })

            await loadGroups()
            setDetailReloadToken((value) => value + 1)
            showNotice(
                role === 2
                    ? '已设为管理员'
                    : '已取消管理员',
            )
        } catch (err) {
            showNotice(
                err.message || '修改成员角色失败',
                'error',
            )
        } finally {
            setActionKey('')
        }
    }

    async function handleTransferOwner(member) {
        const profile = profiles[member.user_id]
        const name = profile?.nickname || member.user_id

        if (
            !window.confirm(
                `确定将群主转让给「${name}」吗？转让后你将变成普通成员。`,
            )
        ) {
            return
        }

        setActionKey(`transfer-${member.user_id}`)
        setMemberMenuAnchor(null)

        try {
            await transferGroupOwnership({
                groupId: selectedID,
                newOwnerId: member.user_id,
            })

            await loadGroups()
            setDetailReloadToken((value) => value + 1)
            showNotice('群主已转让')
        } catch (err) {
            showNotice(
                err.message || '转让群主失败',
                'error',
            )
        } finally {
            setActionKey('')
        }
    }

    async function handleDissolveGroup() {
        if (!groupDetail) {
            return
        }

        if (
            !window.confirm(
                `确定解散群组「${groupDetail.name}」吗？此操作不可恢复。`,
            )
        ) {
            return
        }

        setActionKey('dissolve')

        try {
            await dissolveGroup(selectedID)
            await loadGroups()
            showNotice('群组已解散')
        } catch (err) {
            showNotice(
                err.message || '解散群组失败',
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
                bgcolor: '#f4f6f9',
                overflow: 'hidden',
            }}
        >
            <Box
                sx={{
                    width: 310,
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
                        display: 'flex',
                        alignItems: 'center',
                    }}
                >
                    <Typography
                        sx={{
                            fontSize: 20,
                            fontWeight: 700,
                        }}
                    >
                        群组
                    </Typography>

                    <Box sx={{ flex: 1 }} />

                    <IconButton
                        size="small"
                        onClick={() => {
                            void loadGroups()
                        }}
                        disabled={loadingGroups}
                    >
                        <RefreshRoundedIcon />
                    </IconButton>

                    <IconButton
                        size="small"
                        onClick={() => setDialogOpen(true)}
                    >
                        <AddRoundedIcon />
                    </IconButton>
                </Box>

                {error && (
                    <Alert severity="error" sx={{ mx: 2, mb: 1 }}>
                        {error}
                    </Alert>
                )}

                <List
                    disablePadding
                    className="chat-scroll"
                    sx={{
                        flex: 1,
                        minHeight: 0,
                        overflowY: 'auto',
                        px: 1,
                        pb: 1,
                    }}
                >
                    {loadingGroups && groups.length === 0 && (
                        <Box
                            sx={{
                                minHeight: 200,
                                display: 'grid',
                                placeItems: 'center',
                            }}
                        >
                            <CircularProgress size={28} />
                        </Box>
                    )}

                    {!loadingGroups && groups.length === 0 && (
                        <Box
                            sx={{
                                px: 2,
                                py: 6,
                                textAlign: 'center',
                            }}
                        >
                            <Typography sx={{ fontSize: 13, color: '#8c95a3' }}>
                                还没有群组
                            </Typography>
                        </Box>
                    )}

                    {groups.map((group) => (
                        <ListItemButton
                            key={group.group_id}
                            selected={selectedID === group.group_id}
                            onClick={() => setSelectedID(group.group_id)}
                            sx={{
                                minHeight: 70,
                                mb: 0.5,
                                borderRadius: 2,
                                '&.Mui-selected': {
                                    bgcolor: '#e8eefb',
                                },
                            }}
                        >
                            <ListItemAvatar>
                                <Avatar
                                    src={group.avatar_url || undefined}
                                    sx={{
                                        bgcolor: colorFor(group.group_id),
                                        fontWeight: 700,
                                    }}
                                >
                                    {group.name.slice(0, 1)}
                                </Avatar>
                            </ListItemAvatar>

                            <ListItemText
                                primary={group.name}
                                secondary={roleLabel(group.role)}
                                primaryTypographyProps={{
                                    fontSize: 14,
                                    fontWeight: 600,
                                    noWrap: true,
                                }}
                                secondaryTypographyProps={{
                                    fontSize: 11,
                                }}
                            />
                        </ListItemButton>
                    ))}
                </List>
            </Box>

            <Box
                className="chat-scroll"
                sx={{
                    flex: 1,
                    minWidth: 0,
                    overflowY: 'auto',
                    px: { xs: 2, md: 4 },
                    py: 3,
                }}
            >
                {!selectedID && (
                    <Box
                        sx={{
                            height: '100%',
                            display: 'grid',
                            placeItems: 'center',
                            textAlign: 'center',
                        }}
                    >
                        <Box>
                            <GroupsRoundedIcon
                                sx={{ fontSize: 54, color: '#aab3c0' }}
                            />
                            <Typography
                                sx={{
                                    mt: 1,
                                    fontSize: 16,
                                    fontWeight: 600,
                                }}
                            >
                                选择或创建一个群组
                            </Typography>
                        </Box>
                    </Box>
                )}

                {selectedID && (
                    <Box
                        sx={{
                            width: '100%',
                            maxWidth: 900,
                            mx: 'auto',
                        }}
                    >
                        {detailError && (
                            <Alert severity="error" sx={{ mb: 2 }}>
                                {detailError}
                            </Alert>
                        )}

                        {loadingDetail && !groupDetail && (
                            <Box
                                sx={{
                                    minHeight: 260,
                                    display: 'grid',
                                    placeItems: 'center',
                                }}
                            >
                                <CircularProgress size={30} />
                            </Box>
                        )}

                        {groupDetail && (
                            <>
                                <Paper
                                    elevation={0}
                                    sx={{
                                        p: 3,
                                        border: '1px solid #e5e8ed',
                                    }}
                                >
                                    <Box
                                        sx={{
                                            display: 'flex',
                                            alignItems: 'center',
                                            gap: 2,
                                        }}
                                    >
                                        <Avatar
                                            src={groupDetail.avatar_url || undefined}
                                            sx={{
                                                width: 64,
                                                height: 64,
                                                bgcolor: colorFor(
                                                    groupDetail.group_id,
                                                ),
                                                fontSize: 24,
                                                fontWeight: 700,
                                            }}
                                        >
                                            {groupDetail.name.slice(0, 1)}
                                        </Avatar>

                                        <Box sx={{ flex: 1, minWidth: 0 }}>
                                            <Typography
                                                sx={{
                                                    fontSize: 22,
                                                    fontWeight: 750,
                                                }}
                                            >
                                                {groupDetail.name}
                                            </Typography>

                                            <Typography
                                                sx={{
                                                    mt: 0.5,
                                                    color: '#7f8998',
                                                    fontSize: 12,
                                                }}
                                            >
                                                {members.length} 位成员
                                            </Typography>
                                        </Box>

                                        <Chip
                                            label={roleLabel(groupDetail.role)}
                                            color={
                                                groupDetail.role === 1
                                                    ? 'primary'
                                                    : 'default'
                                            }
                                        />
                                        {isOwner() && (
                                            <Button
                                                color="error"
                                                size="small"
                                                disabled={Boolean(actionKey)}
                                                onClick={() => {
                                                    void handleDissolveGroup()
                                                }}
                                            >
                                                {actionKey === 'dissolve'
                                                    ? '解散中'
                                                    : '解散群组'}
                                            </Button>
                                        )}
                                        {groupDetail.role !== 1 && (
                                            <Button
                                                color="error"
                                                size="small"
                                                startIcon={<LogoutRoundedIcon />}
                                                disabled={Boolean(actionKey)}
                                                onClick={() => {
                                                    void handleLeaveGroup()
                                                }}
                                            >
                                                {actionKey === 'leave'
                                                    ? '退出中'
                                                    : '退出群组'}
                                            </Button>
                                        )}
                                    </Box>

                                    <Typography
                                        sx={{
                                            mt: 3,
                                            fontSize: 12,
                                            fontWeight: 700,
                                            color: '#7f8998',
                                        }}
                                    >
                                        群公告
                                    </Typography>

                                    <Typography
                                        sx={{
                                            mt: 0.8,
                                            fontSize: 13,
                                            lineHeight: 1.7,
                                            whiteSpace: 'pre-wrap',
                                        }}
                                    >
                                        {groupDetail.notification || '暂无群公告'}
                                    </Typography>

                                    <Typography
                                        sx={{
                                            mt: 2,
                                            color: '#a0a8b4',
                                            fontFamily: 'monospace',
                                            fontSize: 11,
                                            userSelect: 'all',
                                        }}
                                    >
                                        群 ID: {groupDetail.group_id}
                                    </Typography>
                                </Paper>

                                <Box
                                    sx={{
                                        mt: 3,
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
                                        群成员
                                        <Typography
                                            component="span"
                                            sx={{
                                                ml: 1,
                                                color: '#99a1ae',
                                                fontSize: 12,
                                            }}
                                        >
                                            {members.length}
                                        </Typography>
                                    </Typography>

                                    <Box sx={{ flex: 1 }} />

                                    {canManageGroup() && (
                                        <Button
                                            size="small"
                                            startIcon={<PersonAddAltRoundedIcon />}
                                            onClick={() => setAddMembersOpen(true)}
                                        >
                                            添加成员
                                        </Button>
                                    )}
                                </Box>

                                <Paper
                                    elevation={0}
                                    sx={{
                                        border: '1px solid #e5e8ed',
                                        overflow: 'hidden',
                                    }}
                                >
                                    <List disablePadding>
                                        {members.map((member, index) => {
                                            const profile =
                                                profiles[member.user_id]
                                            const name =
                                                profile?.nickname || member.user_id

                                            return (
                                                <ListItem
                                                    key={member.user_id}
                                                    divider={index < members.length - 1}
                                                >
                                                    <ListItemAvatar>
                                                        <Avatar
                                                            src={
                                                                profile?.avatar_url ||
                                                                undefined
                                                            }
                                                            sx={{ bgcolor: '#2f6fed' }}
                                                        >
                                                            {name.slice(0, 1)}
                                                        </Avatar>
                                                    </ListItemAvatar>

                                                    <ListItemText
                                                        primary={
                                                            member.user_id === currentUser?.id
                                                                ? `${name}（我）`
                                                                : name
                                                        }
                                                        secondary={
                                                            profile?.phone ||
                                                            member.user_id
                                                        }
                                                        primaryTypographyProps={{
                                                            fontSize: 14,
                                                            fontWeight: 600,
                                                        }}
                                                        secondaryTypographyProps={{
                                                            fontSize: 11,
                                                        }}
                                                    />

                                                    <Box
                                                        sx={{
                                                            display: 'flex',
                                                            alignItems: 'center',
                                                            gap: 0.5,
                                                        }}
                                                    >
                                                        <Chip
                                                            size="small"
                                                            label={roleLabel(member.role)}
                                                            variant="outlined"
                                                        />

                                                        {canKickMember(member) && (
                                                            <Tooltip title="移出群组">
                                                                <span>
                                                                    <IconButton
                                                                        size="small"
                                                                        color="error"
                                                                        disabled={Boolean(actionKey)}
                                                                        onClick={() => {
                                                                            void handleKickMember(member)
                                                                        }}
                                                                    >
                                                                        {actionKey ===
                                                                            `kick-${member.user_id}` ? (
                                                                            <CircularProgress size={17} />
                                                                        ) : (
                                                                            <DeleteOutlineRoundedIcon fontSize="small" />
                                                                        )}
                                                                    </IconButton>
                                                                </span>
                                                            </Tooltip>
                                                        )}
                                                        {canManageMemberRole(member) && (
                                                            <IconButton
                                                                size="small"
                                                                disabled={Boolean(actionKey)}
                                                                onClick={(event) => {
                                                                    setMemberMenuAnchor(event.currentTarget)
                                                                    setMemberMenuTarget(member)
                                                                }}
                                                            >
                                                                <MoreVertRoundedIcon fontSize="small" />
                                                            </IconButton>
                                                        )}
                                                    </Box>
                                                </ListItem>
                                            )
                                        })}
                                    </List>
                                </Paper>
                            </>
                        )}
                    </Box>
                )}
            </Box>

            <CreateGroupDialog
                open={dialogOpen}
                friends={friends}
                onClose={() => setDialogOpen(false)}
                onCreated={handleCreated}
            />

            <AddGroupMembersDialog
                open={addMembersOpen}
                groupId={selectedID}
                friends={friends}
                existingMemberIDs={members.map(
                    (member) => member.user_id,
                )}
                onClose={() => setAddMembersOpen(false)}
                onAdded={async () => {
                    await loadGroups()
                    setDetailReloadToken((value) => value + 1)
                    showNotice('群成员已添加')
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
                >
                    {notice.message}
                </Alert>
            </Snackbar>
            <Menu
                anchorEl={memberMenuAnchor}
                open={Boolean(memberMenuAnchor)}
                onClose={() => {
                    setMemberMenuAnchor(null)
                    setMemberMenuTarget(null)
                }}
            >
                {memberMenuTarget?.role === 3 && (
                    <MenuItem
                        onClick={() => {
                            void handleChangeRole(memberMenuTarget, 2)
                        }}
                    >
                        设为管理员
                    </MenuItem>
                )}

                {memberMenuTarget?.role === 2 && (
                    <MenuItem
                        onClick={() => {
                            void handleChangeRole(memberMenuTarget, 3)
                        }}
                    >
                        取消管理员
                    </MenuItem>
                )}

                <MenuItem
                    onClick={() => {
                        void handleTransferOwner(memberMenuTarget)
                    }}
                >
                    转让群主
                </MenuItem>
            </Menu>
        </Box>
    )
}
