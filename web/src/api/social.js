import { request } from './http'

export function listFriends() {
  return request('/v1/social/friends')
}

export function listIncomingFriendRequests() {
  return request('/v1/social/friend/requests/incoming')
}

export function sendFriendRequest({
  recipientId,
  message,
}) {
  return request('/v1/social/friend/requests', {
    method: 'POST',
    body: {
      recipient_id: recipientId,
      message,
    },
  })
}

export function handleFriendRequest({
  requestId,
  accept,
}) {
  return request('/v1/social/friend/requests', {
    method: 'PUT',
    body: {
      request_id: requestId,
      accept,
    },
  })
}

export function removeFriend(friendId) {
  return request(
    `/v1/social/friends/${encodeURIComponent(friendId)}`,
    {
      method: 'DELETE',
    },
  )
}

export function listGroups() {
  return request('/v1/social/groups')
}

export function createGroup({
  name,
  avatarUrl = '',
  notification = '',
  memberIds = [],
}) {
  return request('/v1/social/groups', {
    method: 'POST',
    body: {
      name,
      avatar_url: avatarUrl,
      notification,
      member_ids: memberIds,
    },
  })
}

export function getGroup(groupId) {
  return request(
    `/v1/social/groups/${encodeURIComponent(groupId)}`,
  )
}

export function listGroupMembers(groupId) {
  return request(
    `/v1/social/groups/${encodeURIComponent(
      groupId,
    )}/members`,
  )
}

export function addGroupMembers(
  groupId,
  memberIds,
) {
  return request(
    `/v1/social/groups/${encodeURIComponent(
      groupId,
    )}/members`,
    {
      method: 'POST',
      body: {
        member_ids: memberIds,
      },
    },
  )
}

export function kickGroupMember(
  groupId,
  userId,
) {
  return request(
    `/v1/social/groups/${encodeURIComponent(
      groupId,
    )}/members/${encodeURIComponent(userId)}`,
    {
      method: 'DELETE',
    },
  )
}

export function leaveGroup(groupId) {
  return request(
    `/v1/social/groups/${encodeURIComponent(
      groupId,
    )}/leave`,
    {
      method: 'POST',
    },
  )
}

export function changeGroupMemberRole({
  groupId,
  userId,
  role,
}) {
  return request(
    `/v1/social/groups/${encodeURIComponent(
      groupId,
    )}/members/${encodeURIComponent(
      userId,
    )}/role`,
    {
      method: 'PUT',
      body: {
        role,
      },
    },
  )
}

export function transferGroupOwnership({
  groupId,
  newOwnerId,
}) {
  return request(
    `/v1/social/groups/${encodeURIComponent(
      groupId,
    )}/owner`,
    {
      method: 'PUT',
      body: {
        new_owner_id: newOwnerId,
      },
    },
  )
}

export function dissolveGroup(groupId) {
  return request(
    `/v1/social/groups/${encodeURIComponent(
      groupId,
    )}`,
    {
      method: 'DELETE',
    },
  )
}