import { openDB } from 'idb'

const DB_NAME = 'easy-chat-cache'
const DB_VERSION = 2

const dbPromise = openDB(DB_NAME, DB_VERSION, {
    upgrade(db) {
        if (!db.objectStoreNames.contains('messages')) {
            const messageStore = db.createObjectStore(
                'messages',
                {
                    keyPath: 'cache_key',
                },
            )

            messageStore.createIndex(
                'conversation_id',
                'conversation_id',
            )
        }

        if (!db.objectStoreNames.contains('sync_cursors')) {
            db.createObjectStore('sync_cursors', {
                keyPath: 'conversation_id',
            })
        }

        if (!db.objectStoreNames.contains('pending_messages')) {
            const pendingStore = db.createObjectStore(
                'pending_messages',
                {
                    keyPath: 'client_message_id',
                },
            )

            pendingStore.createIndex(
                'conversation_id',
                'conversation_id',
            )
        }
    },
})

function messageCacheKey(message) {
    return (
        message.message_id ||
        [
            message.conversation_id,
            message.client_message_id,
        ].join(':')
    )
}

function sortMessages(messages) {
    return [...messages].sort((left, right) => {
        const leftSequence = Number(left.sequence || 0)
        const rightSequence = Number(right.sequence || 0)

        if (leftSequence > 0 && rightSequence > 0) {
            return leftSequence - rightSequence
        }

        if (leftSequence > 0) {
            return 1
        }

        if (rightSequence > 0) {
            return -1
        }

        return (
            Number(left.created_at || 0) -
            Number(right.created_at || 0)
        )
    })
}

export async function loadCachedMessages(
    conversationID,
) {
    const db = await dbPromise

    const rows = await db.getAllFromIndex(
        'messages',
        'conversation_id',
        conversationID,
    )

    return sortMessages(rows)
}

export async function cacheMessages(messages) {
    if (!Array.isArray(messages) || messages.length === 0) {
        return
    }

    const db = await dbPromise
    const transaction = db.transaction(
        'messages',
        'readwrite',
    )

    try {
        for (const message of messages) {
            if (
                !message.conversation_id ||
                !message.content
            ) {
                continue
            }

            await transaction.store.put({
                ...message,
                cache_key: messageCacheKey(message),
            })
        }

        await transaction.done
    } catch (error) {
        transaction.abort()
        throw error
    }
}

export async function getSyncCursor(
    conversationID,
) {
    const db = await dbPromise
    const cursor = await db.get(
        'sync_cursors',
        conversationID,
    )

    return Number(cursor?.sequence || 0)
}

export async function saveSyncCursor(
    conversationID,
    sequence,
) {
    const value = Number(sequence || 0)

    if (!conversationID || value <= 0) {
        return
    }

    const db = await dbPromise
    const existing = await db.get(
        'sync_cursors',
        conversationID,
    )

    if (
        existing &&
        Number(existing.sequence || 0) >= value
    ) {
        return
    }

    await db.put('sync_cursors', {
        conversation_id: conversationID,
        sequence: value,
        updated_at: Date.now(),
    })
}

export async function queuePendingMessage(message) {
    if (
        !message.conversation_id ||
        !message.client_message_id ||
        !message.content
    ) {
        throw new Error('invalid pending message')
    }

    const db = await dbPromise
    const existing = await db.get(
        'pending_messages',
        message.client_message_id,
    )

    await db.put('pending_messages', {
        ...existing,
        ...message,
        deliveryStatus: 'pending',
        retry_count: Number(existing?.retry_count || 0),
        queued_at: existing?.queued_at || Date.now(),
        updated_at: Date.now(),
    })
}

export async function listPendingMessages(
    conversationID,
) {
    const db = await dbPromise

    const rows = await db.getAllFromIndex(
        'pending_messages',
        'conversation_id',
        conversationID,
    )

    return rows.sort(
        (left, right) =>
            Number(left.queued_at || 0) -
            Number(right.queued_at || 0),
    )
}

export async function listAllPendingMessages() {
    const db = await dbPromise
    const rows = await db.getAll('pending_messages')

    return rows.sort(
        (left, right) =>
            Number(left.queued_at || 0) -
            Number(right.queued_at || 0),
    )
}

export async function updatePendingMessage(
    clientMessageID,
    changes,
) {
    const db = await dbPromise
    const existing = await db.get(
        'pending_messages',
        clientMessageID,
    )

    if (!existing) {
        return
    }

    await db.put('pending_messages', {
        ...existing,
        ...changes,
        updated_at: Date.now(),
    })
}

export async function removePendingMessage(
    clientMessageID,
) {
    if (!clientMessageID) {
        return
    }

    const db = await dbPromise
    await db.delete(
        'pending_messages',
        clientMessageID,
    )
}