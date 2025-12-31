import request from '@/utils/request'

// 1. 发送对话 (支持会话ID)
export function chat(question, conversationId) {
    return request({
        url: '/chat',
        method: 'post',
        data: {
            question,
            conversation_id: conversationId
        }
    })
}

// 2. 获取会话列表 (对应左侧边栏)
export function getConversationList() {
    return request({
        url: '/history',
        method: 'get'
    })
}

// 3. 获取某个会话的详细消息
export function getConversationMessages(id) {
    return request({
        url: '/history/messages',
        method: 'get',
        params: { conversation_id: id }
    })
}

// 4. 重命名会话
export function renameConversation(id, title) {
    return request({
        url: '/history/rename',
        method: 'post',
        data: { id, title }
    })
}

// 5. 删除会话
export function deleteConversation(id) {
    return request({
        url: '/history/delete',
        method: 'post',
        data: { id }
    })
}