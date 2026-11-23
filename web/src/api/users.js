import { request } from './http'

export async function getUsers(userIds) {
  const uniqueIDs = [...new Set(
    userIds.filter(Boolean),
  )]

  if (uniqueIDs.length === 0) {
    return []
  }

  const data = await request(
    `/v1/user/batch?user_ids=${encodeURIComponent(
      uniqueIDs.join(','),
    )}`,
  )

  return data.list || []
}