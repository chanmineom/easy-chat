import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import {
  Box,
  Typography,
} from '@mui/material'

import {
  listAllPendingMessages,
  removePendingMessage,
  updatePendingMessage,
} from '../storage/chatCache'

import NavigationRail from '../components/NavigationRail'
import ConversationSidebar from '../components/ConversationSidebar'
import ContactsPanel from '../components/ContactsPanel'
import { listConversations } from '../api/im'
import { listGroups } from '../api/social'
import { useAuthStore } from '../stores/authStore'
import MessagePanel from '../components/MessagePanel'
import { chatSocket } from '../realtime/chatSocket'
import GroupsPanel from '../components/GroupsPanel'

const avatarColors = [
  '#2f6fed',
  '#2a9d8f',
  '#ef8354',
  '#8b5cf6',
  '#d65a7a',
  '#4f6d7a',
]

function colorFor(value) {
  let total = 0

  for (const char of value || '') {
    total += char.charCodeAt(0)
  }

  return avatarColors[total % avatarColors.length]
}

function formatConversationTime(timestamp) {
  if (!timestamp) {
    return ''
  }

  const date = new Date(timestamp)
  const now = new Date()

  if (date.toDateString() === now.toDateString()) {
    return new Intl.DateTimeFormat('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
    }).format(date)
  }

  return new Intl.DateTimeFormat('zh-CN', {
    month: 'numeric',
    day: 'numeric',
  }).format(date)
}

function normalizeConversation(
  item,
  currentUserID,
  groupByID,
) {
  const lastMessage = item.last_message
  const group =
    item.chat_type === 1
      ? groupByID[item.conversation_id]
      : null
  let preview = '暂无消息'

  if (lastMessage?.content) {
    preview = lastMessage.content
  } else if (lastMessage?.content_type) {
    preview = `[${lastMessage.content_type}]`
  }

  if (
    item.chat_type === 1 &&
    lastMessage?.sender_id &&
    lastMessage.sender_id !== currentUserID &&
    item.last_sender_name
  ) {
    preview = `${item.last_sender_name}: ${preview}`
  }

  const fallbackName =
    item.title && !item.title.startsWith('u_')
      ? item.title
      : item.chat_type === 1
        ? '群聊'
        : '未命名会话'

  return {
    id: item.conversation_id,
    chatType: item.chat_type,
    name: group?.name || fallbackName,
    avatarUrl: group?.avatar_url || item.avatar_url,
    preview,
    time: formatConversationTime(item.last_activity_at),
    unread: item.unread_count || 0,
    color: colorFor(item.conversation_id),
    lastSeq: item.last_seq || 0,
    raw: item,
  }
}

export default function ChatLayout() {
  const currentUser = useAuthStore((state) => state.user)

  const [activeNavigation, setActiveNavigation] =
    useState('chat')
  const [conversationData, setConversationData] =
    useState([])
  const [groupByID, setGroupByID] = useState({})
  const [selectedID, setSelectedID] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [socketStatus, setSocketStatus] = useState('idle')
  const [pageVisible, setPageVisible] = useState(
    document.visibilityState === 'visible',
  )

  const loadConversations = useCallback(async () => {
    setLoading(true)
    setError('')

    try {
      const data = await listConversations({
        limit: 50,
      })

      setConversationData(data.list || [])
    } catch (err) {
      setError(err.message || '会话加载失败')
    } finally {
      setLoading(false)
    }
  }, [])

  const loadGroups = useCallback(async () => {
    try {
      const data = await listGroups()

      setGroupByID(
        Object.fromEntries(
          (data.list || []).map(
            (group) => [group.group_id, group],
          ),
        ),
      )
    } catch (err) {
      console.error('加载群组名称失败', err)
    }
  }, [])

  const flushPendingMessages = useCallback(async () => {
    if (
      flushingPending.current ||
      socketStatus !== 'connected'
    ) {
      return
    }

    flushingPending.current = true

    try {
      const pendingMessages =
        await listAllPendingMessages()

      for (const message of pendingMessages) {
        if (
          !message.conversation_id ||
          !message.client_message_id ||
          !message.content
        ) {
          continue
        }

        await updatePendingMessage(
          message.client_message_id,
          {
            deliveryStatus: 'sending',
            retry_count:
              Number(message.retry_count || 0) + 1,
          },
        )

        try {
          chatSocket.sendMessage({
            conversationId:
              message.conversation_id,
            clientMessageId:
              message.client_message_id,
            content: message.content,
          })
        } catch {
          await updatePendingMessage(
            message.client_message_id,
            {
              deliveryStatus: 'pending',
            },
          )
          break
        }
      }
    } finally {
      flushingPending.current = false
    }
  }, [socketStatus])

  useEffect(() => {
    if (socketStatus === 'connected') {
      void flushPendingMessages()
    }
  }, [flushPendingMessages, socketStatus])

  useEffect(() => {
    const unsubscribeStatus =
      chatSocket.subscribeStatus(setSocketStatus)

    chatSocket.connect()

    return () => {
      unsubscribeStatus()
      chatSocket.disconnect()
    }
  }, [])

  useEffect(() => {
    void loadConversations()
    void loadGroups()

    const timer = window.setInterval(() => {
      void loadConversations()
      void loadGroups()
    }, 10000)

    return () => {
      window.clearInterval(timer)
    }
  }, [loadConversations, loadGroups])

  useEffect(() => {
    function handleVisibilityChange() {
      setPageVisible(
        document.visibilityState === 'visible',
      )
    }

    document.addEventListener(
      'visibilitychange',
      handleVisibilityChange,
    )

    return () => {
      document.removeEventListener(
        'visibilitychange',
        handleVisibilityChange,
      )
    }
  }, [])

  useEffect(() => {
    let active = true
    const reloadTimers = new Set()

    const unsubscribe = chatSocket.subscribe((event) => {
      if (event.type === 'connection.ready') {
        void loadConversations()
        void flushPendingMessages()
        return
      }

      if (event.type !== 'message.created' || event.type === 'message.rejected') {
        const clientMessageID =
          event.data?.client_message_id

        if (clientMessageID) {
          void removePendingMessage(
            clientMessageID,
          )
        }
      }

      if (event.type !== 'message.created') {
        return
      }

      const message = event.data

      if (
        !message ||
        !message.conversation_id ||
        message.sequence <= 0
      ) {
        return
      }

      const sentByMe =
        message.sender_id === currentUser?.id

      if (!sentByMe) {
        try {
          chatSocket.sendReceipt({
            type: 'message.delivered',
            conversationId: message.conversation_id,
            sequence: message.sequence,
          })
        } catch {
          // 连接刚断开时忽略，重连后会继续同步。
        }
      }

      const viewingConversation =
        activeNavigation === 'chat' &&
        selectedID === message.conversation_id &&
        pageVisible

      if (viewingConversation) {
        try {
          chatSocket.sendReceipt({
            type: 'message.read',
            conversationId: message.conversation_id,
            sequence: message.sequence,
          })
        } catch {
          // 连接刚断开时忽略。
        }

        setConversationData((current) =>
          current.map((item) => {
            if (
              item.conversation_id !==
              message.conversation_id
            ) {
              return item
            }

            return {
              ...item,
              unread_count: 0,
              read_seq: Math.max(
                Number(item.read_seq || 0),
                message.sequence,
              ),
            }
          }),
        )
      }

      const timer = window.setTimeout(() => {
        reloadTimers.delete(timer)

        if (active) {
          void loadConversations()
        }
      }, 500)

      reloadTimers.add(timer)
    })

    return () => {
      active = false
      unsubscribe()

      for (const timer of reloadTimers) {
        window.clearTimeout(timer)
      }

      reloadTimers.clear()
    }
  }, [
    activeNavigation,
    currentUser?.id,
    flushPendingMessages,
    loadConversations,
    pageVisible,
    selectedID,
  ])

  const conversations = useMemo(
    () =>
      conversationData.map((item) =>
        normalizeConversation(
          item,
          currentUser?.id,
          groupByID,
        ),
      ),
    [conversationData, currentUser?.id, groupByID],
  )

  useEffect(() => {
    if (conversations.length === 0) {
      setSelectedID('')
      return
    }

    const exists = conversations.some(
      (conversation) => conversation.id === selectedID,
    )

    if (!exists) {
      setSelectedID(conversations[0].id)
    }
  }, [conversations, selectedID])

  const selectedConversation = useMemo(
    () =>
      conversations.find(
        (conversation) => conversation.id === selectedID,
      ),
    [conversations, selectedID],
  )

  useEffect(() => {
    if (
      activeNavigation !== 'chat' ||
      socketStatus !== 'connected' ||
      !pageVisible ||
      !selectedConversation?.id ||
      !selectedConversation.lastSeq
    ) {
      return
    }

    try {
      chatSocket.sendReceipt({
        type: 'message.read',
        conversationId: selectedConversation.id,
        sequence: selectedConversation.lastSeq,
      })
    } catch {
      return
    }

    setConversationData((current) =>
      current.map((item) => {
        if (
          item.conversation_id !== selectedConversation.id
        ) {
          return item
        }

        return {
          ...item,
          unread_count: 0,
          read_seq: Math.max(
            Number(item.read_seq || 0),
            selectedConversation.lastSeq,
          ),
        }
      }),
    )
  }, [
    activeNavigation,
    pageVisible,
    selectedConversation?.id,
    selectedConversation?.lastSeq,
    socketStatus,
  ])

  return (
    <Box
      sx={{
        width: '100%',
        height: '100%',
        display: 'flex',
        overflow: 'hidden',
      }}
    >
      <NavigationRail
        active={activeNavigation}
        onChange={setActiveNavigation}
      />

      {activeNavigation === 'chat' && (
        <>
          <ConversationSidebar
            conversations={conversations}
            selectedID={selectedID}
            onSelect={setSelectedID}
            loading={loading}
            error={error}
            onRetry={() => {
              void loadConversations()
            }}
          />

          <MessagePanel
            key={selectedConversation?.id || 'empty'}
            conversation={selectedConversation}
            socketStatus={socketStatus}
          />
        </>
      )}

      {activeNavigation === 'contacts' && (
        <ContactsPanel />
      )}

      {activeNavigation === 'groups' && (
        <GroupsPanel />
      )}
    </Box>
  )
}
