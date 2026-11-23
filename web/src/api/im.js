import { request } from './http'

export function listConversations({
  limit = 50,
  beforeActivityAt = 0,
  beforeConversationId = '',
} = {}) {
  const params = new URLSearchParams({
    limit: String(limit),
  })

  if (beforeActivityAt > 0) {
    params.set(
      'before_activity_at',
      String(beforeActivityAt),
    )
  }

  if (beforeConversationId) {
    params.set(
      'before_conversation_id',
      beforeConversationId,
    )
  }

  return request(
    `/v1/im/conversations?${params.toString()}`,
  )
}

export function syncMessages({
  conversationId,
  afterSeq = 0,
  beforeSeq = 0,
  limit = 50,
}) {
  const params = new URLSearchParams({
    conversation_id: conversationId,
    after_seq: String(afterSeq),
    before_seq: String(beforeSeq),
    limit: String(limit),
  })

  return request(
    `/v1/im/messages?${params.toString()}`,
  )
}

export function issueWSTicket(deviceId) {
  return request('/v1/im/ws/ticket', {
    method: 'POST',
    body: {
      device_id: deviceId,
    },
  })
}