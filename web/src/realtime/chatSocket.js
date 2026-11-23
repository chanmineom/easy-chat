import { issueWSTicket } from '../api/im'

const DEVICE_ID_KEY = 'easy-chat-device-id'
const MAX_RECONNECT_DELAY = 30000
const HEARTBEAT_INTERVAL = 10000
const PONG_TIMEOUT = 5000

function createID() {
    if (globalThis.crypto?.randomUUID) {
        return globalThis.crypto.randomUUID()
    }

    return [
        Date.now().toString(36),
        Math.random().toString(36).slice(2),
    ].join('-')
}

function getDeviceID() {
    const saved = window.localStorage.getItem(DEVICE_ID_KEY)

    if (saved) {
        return saved
    }

    const deviceID = `web-${createID()}`
    window.localStorage.setItem(DEVICE_ID_KEY, deviceID)
    return deviceID
}

function buildWebSocketURL(ticket) {
    const protocol =
        window.location.protocol === 'https:' ? 'wss:' : 'ws:'

    return (
        `${protocol}//${window.location.host}/ws` +
        `?ticket=${encodeURIComponent(ticket)}`
    )
}

class ChatSocket {
    constructor() {
        this.socket = null
        this.status = 'idle'
        this.eventListeners = new Set()
        this.statusListeners = new Set()

        this.reconnectTimer = null
        this.heartbeatTimer = null
        this.pongTimeoutTimer = null
        this.reconnectAttempt = 0
        this.connecting = false
        this.manuallyClosed = true
        this.generation = 0
        this.lastPongAt = 0

        window.addEventListener('offline', () => {
            if (this.manuallyClosed) {
                return
            }

            this.setStatus('disconnected')

            if (this.socket) {
                this.socket.close()
            } else {
                this.scheduleReconnect(this.generation)
            }
        })

        window.addEventListener('online', () => {
            if (
                this.manuallyClosed ||
                this.connecting ||
                (
                    this.socket &&
                    this.socket.readyState === WebSocket.OPEN
                )
            ) {
                return
            }

            window.clearTimeout(this.reconnectTimer)
            this.reconnectAttempt = 0
            void this.open(this.generation)
        })
    }

    connect() {
        this.manuallyClosed = false

        if (
            this.socket &&
            (
                this.socket.readyState === WebSocket.OPEN ||
                this.socket.readyState === WebSocket.CONNECTING
            )
        ) {
            return
        }

        if (this.connecting) {
            return
        }

        this.generation += 1
        this.reconnectAttempt = 0

        void this.open(this.generation)
    }

    disconnect() {
        this.manuallyClosed = true
        this.generation += 1
        this.connecting = false
        this.reconnectAttempt = 0

        this.clearTimers()

        if (this.socket) {
            this.socket.onopen = null
            this.socket.onmessage = null
            this.socket.onerror = null
            this.socket.onclose = null
            this.socket.close(1000, 'client closed')
            this.socket = null
        }

        this.setStatus('closed')
    }

    sendMessage({
        conversationId,
        clientMessageId,
        content,
    }) {
        if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
            throw new Error('WebSocket 尚未连接')
        }

        const requestID = createID()

        this.socket.send(JSON.stringify({
            type: 'message.send',
            request_id: requestID,
            data: {
                conversation_id: conversationId,
                client_message_id: clientMessageId,
                content_type: 'text',
                content,
            },
        }))

        return requestID
    }

    sendReceipt({
        type,
        conversationId,
        sequence,
    }) {
        if (
            type !== 'message.delivered' &&
            type !== 'message.read'
        ) {
            throw new Error('unsupported receipt type')
        }

        if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
            throw new Error('WebSocket 尚未连接')
        }

        if (!conversationId || sequence <= 0) {
            return ''
        }

        const requestID = createID()

        this.socket.send(JSON.stringify({
            type,
            request_id: requestID,
            data: {
                conversation_id: conversationId,
                sequence,
            },
        }))

        return requestID
    }

    subscribe(listener) {
        this.eventListeners.add(listener)

        return () => {
            this.eventListeners.delete(listener)
        }
    }

    subscribeStatus(listener) {
        this.statusListeners.add(listener)
        listener(this.status)

        return () => {
            this.statusListeners.delete(listener)
        }
    }

    async open(generation) {
        if (
            this.manuallyClosed ||
            generation !== this.generation
        ) {
            return
        }

        this.connecting = true
        this.setStatus('connecting')

        try {
            const data = await issueWSTicket(getDeviceID())

            if (
                this.manuallyClosed ||
                generation !== this.generation
            ) {
                return
            }

            const socket = new WebSocket(
                buildWebSocketURL(data.ticket),
            )

            this.socket = socket

            socket.onopen = () => {
                if (
                    this.manuallyClosed ||
                    generation !== this.generation
                ) {
                    socket.close()
                    return
                }

                this.reconnectAttempt = 0
                this.lastPongAt = Date.now()
                this.setStatus('connected')
                this.startHeartbeat(socket)
            }

            socket.onmessage = (message) => {
                let envelope

                try {
                    envelope = JSON.parse(message.data)
                } catch {
                    this.emit({
                        type: 'error',
                        error: '收到无法解析的 WebSocket 消息',
                    })
                    return
                }

                if (
                    generation !== this.generation ||
                    this.manuallyClosed
                ) {
                    return
                }

                if (envelope.type === 'pong') {
                    this.lastPongAt = Date.now()
                    window.clearTimeout(this.pongTimeoutTimer)
                    this.pongTimeoutTimer = null
                }

                this.emit(envelope)
            }

            socket.onerror = () => {
                socket.close()
            }

            socket.onclose = () => {
                if (this.socket === socket) {
                    this.socket = null
                }

                this.stopHeartbeat()

                if (
                    generation !== this.generation ||
                    this.manuallyClosed
                ) {
                    return
                }

                this.setStatus('disconnected')
                this.scheduleReconnect(generation)
            }
        } catch (error) {
            if (
                generation !== this.generation ||
                this.manuallyClosed
            ) {
                return
            }

            this.setStatus('disconnected')
            this.emit({
                type: 'error',
                error:
                    error.message || 'WebSocket 连接失败',
            })
            this.scheduleReconnect(generation)
        } finally {
            if (generation === this.generation) {
                this.connecting = false
            }
        }
    }

    scheduleReconnect(generation) {
        if (
            this.manuallyClosed ||
            generation !== this.generation
        ) {
            return
        }

        window.clearTimeout(this.reconnectTimer)

        const exponentialDelay = Math.min(
            MAX_RECONNECT_DELAY,
            1000 * (2 ** this.reconnectAttempt),
        )
        const jitter = Math.random() * 500

        this.reconnectAttempt += 1

        this.reconnectTimer = window.setTimeout(() => {
            if (
                this.manuallyClosed ||
                generation !== this.generation
            ) {
                return
            }

            void this.open(generation)
        }, exponentialDelay + jitter)
    }

    startHeartbeat(socket) {
        this.stopHeartbeat()

        this.heartbeatTimer = window.setInterval(() => {
            if (socket.readyState !== WebSocket.OPEN) {
                return
            }

            const sentAt = Date.now()

            window.clearTimeout(this.pongTimeoutTimer)

            socket.send(JSON.stringify({
                type: 'ping',
                request_id: createID(),
            }))

            this.pongTimeoutTimer = window.setTimeout(() => {
                if (
                    socket.readyState !== WebSocket.OPEN ||
                    this.lastPongAt >= sentAt
                ) {
                    return
                }

                this.setStatus('disconnected')
                socket.close()
            }, PONG_TIMEOUT)
        }, HEARTBEAT_INTERVAL)
    }

    stopHeartbeat() {
        if (this.heartbeatTimer) {
            window.clearInterval(this.heartbeatTimer)
            this.heartbeatTimer = null
        }

        if (this.pongTimeoutTimer) {
            window.clearTimeout(this.pongTimeoutTimer)
            this.pongTimeoutTimer = null
        }
    }

    clearTimers() {
        window.clearTimeout(this.reconnectTimer)
        this.reconnectTimer = null
        this.stopHeartbeat()
    }

    emit(envelope) {
        for (const listener of this.eventListeners) {
            try {
                listener(envelope)
            } catch (error) {
                console.error('WebSocket event listener failed', error)
            }
        }
    }

    setStatus(status) {
        if (this.status === status) {
            return
        }

        this.status = status

        for (const listener of this.statusListeners) {
            try {
                listener(status)
            } catch (error) {
                console.error('WebSocket status listener failed', error)
            }
        }
    }
}

export const chatSocket = new ChatSocket()
export { createID }
