import { defineStore } from 'pinia'
import axios from 'axios'
import { ref, computed } from 'vue'
import { useUserStore } from './user'

export const useMessageStore = defineStore('message', () => {
  const userStore = useUserStore()

  // 状态
  const recentList = ref([])               // 最近联系列表
  const friendList = ref([])               // 好友列表
  const groupList = ref([])                // 群聊列表
  const blacklist = ref([])                // 黑名单
  const chatMessages = ref([])             // 当前聊天消息列表
  const currentChatId = ref('')            // 当前聊天ID（好友ID或群聊ID）
  const unreadCounts = ref({})             // 未读消息计数 { [chatId]: count }

  // 计算属性：总未读消息数
  const totalUnreadCount = computed(() => {
    return Object.values(unreadCounts.value).reduce((sum, count) => sum + count, 0)
  })

  // 计算属性：分类未读消息数
  const recentUnreadCount = computed(() => {
    return recentList.value.reduce((sum, item) => sum + (item.unreadCount || 0), 0)
  })

  const friendUnreadCount = computed(() => {
    return friendList.value.reduce((sum, friend) => sum + (friend.unreadCount || 0), 0)
  })

  const groupUnreadCount = computed(() => {
    return groupList.value.reduce((sum, group) => sum + (group.unreadCount || 0), 0)
  })

  // 获取最近联系列表
  const fetchRecentList = async () => {
    try {
      const res = await axios.get('/api/messages/recent')
      if (res.data.code === 200) {
        recentList.value = res.data.data
      }
    } catch (error) {
      console.error('获取最近联系列表失败：', error)
    }
  }

  // 获取好友列表
  const fetchFriendList = async () => {
    try {
      const res = await axios.get('/api/contacts/friends')
      if (res.data.code === 200) {
        friendList.value = res.data.data
      }
    } catch (error) {
      console.error('获取好友列表失败：', error)
    }
  }

  // 获取群聊列表
  const fetchGroupList = async () => {
    try {
      const res = await axios.get('/api/contacts/groups')
      if (res.data.code === 200) {
        groupList.value = res.data.data
      }
    } catch (error) {
      console.error('获取群聊列表失败：', error)
    }
  }

  // 获取黑名单
  const fetchBlacklist = async () => {
    try {
      const res = await axios.get('/api/contacts/blacklist')
      if (res.data.code === 200) {
        blacklist.value = res.data.data
      }
    } catch (error) {
      console.error('获取黑名单失败：', error)
    }
  }

  // 获取聊天记录
  const fetchChatMessages = async (chatId, isGroup = false) => {
    try {
      const url = isGroup 
        ? `/api/messages/group/${chatId}` 
        : `/api/messages/friend/${chatId}`
      
      const res = await axios.get(url)
      if (res.data.code === 200) {
        chatMessages.value = res.data.data
        currentChatId.value = chatId
        // 标记已读
        markAsRead(chatId, isGroup)
      }
    } catch (error) {
      console.error('获取聊天记录失败：', error)
    }
  }

  // 发送消息
  const sendMessage = async (chatId, content, contentType = 1, isGroup = false) => {
    try {
      const res = await axios.post('/api/messages/send', {
        targetId: chatId,
        content,
        contentType,
        isGroup
      })
      if (res.data.code === 200) {
        // 本地添加消息
        chatMessages.value.push(res.data.data)
        return true
      }
      return false
    } catch (error) {
      console.error('发送消息失败：', error)
      return false
    }
  }

  // 撤回消息
  const recallMessage = async (msgId) => {
    try {
      const res = await axios.post('/api/messages/recall', {
        msgId
      })
      if (res.data.code === 200) {
        // 本地更新消息状态
        const msgIndex = chatMessages.value.findIndex(msg => msg.msg_id === msgId)
        if (msgIndex !== -1) {
          chatMessages.value[msgIndex].is_recalled = true
        }
        return true
      }
      return false
    } catch (error) {
      console.error('撤回消息失败：', error)
      return false
    }
  }

  // 标记消息为已读
  const markAsRead = async (chatId, isGroup = false) => {
    try {
      await axios.post('/api/messages/read', {
        targetId: chatId,
        isGroup
      })
      // 更新未读计数
      unreadCounts.value[chatId] = 0
    } catch (error) {
      console.error('标记已读失败：', error)
    }
  }

  // 搜索聊天记录
  const searchChatHistory = async (chatId, keyword, isGroup = false) => {
    try {
      const res = await axios.get('/api/messages/search', {
        params: {
          targetId: chatId,
          keyword,
          isGroup
        }
      })
      return res.data.code === 200 ? res.data.data : []
    } catch (error) {
      console.error('搜索聊天记录失败：', error)
      return []
    }
  }

  // 添加好友到黑名单
  const addToBlacklist = async (fuid) => {
    try {
      const res = await axios.post('/api/contacts/blacklist/add', { fuid })
      if (res.data.code === 200) {
        // 刷新黑名单和好友列表
        await fetchBlacklist()
        await fetchFriendList()
        return true
      }
      return false
    } catch (error) {
      console.error('添加到黑名单失败：', error)
      return false
    }
  }

  // 从黑名单移除
  const removeFromBlacklist = async (fuid) => {
    try {
      const res = await axios.post('/api/contacts/blacklist/remove', { fuid })
      if (res.data.code === 200) {
        await fetchBlacklist()
        await fetchFriendList()
        return true
      }
      return false
    } catch (error) {
      console.error('从黑名单移除失败：', error)
      return false
    }
  }

  // 退出群聊
  const exitGroup = async (groupId) => {
    try {
      const res = await axios.post('/api/groups/exit', { groupId })
      if (res.data.code === 200) {
        await fetchGroupList()
        await fetchRecentList()
        // 如果当前在退出的群聊中，清空聊天记录
        if (currentChatId.value === groupId) {
          chatMessages.value = []
          currentChatId.value = ''
        }
        return true
      }
      return false
    } catch (error) {
      console.error('退出群聊失败：', error)
      return false
    }
  }

  return {
    // 列表数据
    recentList,
    friendList,
    groupList,
    blacklist,
    chatMessages,
    currentChatId,
    // 未读计数
    totalUnreadCount,
    recentUnreadCount,
    friendUnreadCount,
    groupUnreadCount,
    // 方法
    fetchRecentList,
    fetchFriendList,
    fetchGroupList,
    fetchBlacklist,
    fetchChatMessages,
    sendMessage,
    recallMessage,
    markAsRead,
    searchChatHistory,
    addToBlacklist,
    removeFromBlacklist,
    exitGroup
  }
})