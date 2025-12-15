import { defineStore } from 'pinia'
import axios from 'axios'
import { ref } from 'vue'

export const useUserStore = defineStore('user', () => {
  // 状态
  const userInfo = ref({
    fuid: '',         // 用户唯一ID
    nickname: '',     // 昵称
    avatar: '',       // 头像URL
    vipLevel: 0,      // VIP等级
    vipExp: 0,        // VIP经验值
    signature: '',    // 个性签名
    online: false     // 在线状态
  })
  const isLogin = ref(false)       // 登录状态
  const theme = ref('light')       // 主题设置

  // actions：用户登录
  const login = async (username, password) => {
    try {
      const res = await axios.post('/api/auth/login', {
        username,
        password
      })
      if (res.data.code === 200) {
        const userData = res.data.data
        // 更新用户信息
        userInfo.value = {
          fuid: userData.fuid,
          nickname: userData.nickname,
          avatar: userData.avatar,
          vipLevel: userData.vipLevel,
          vipExp: userData.vipExp,
          signature: userData.signature,
          online: true
        }
        isLogin.value = true
        // 保存token到本地存储
        localStorage.setItem('token', userData.token)
        return true
      }
      return false
    } catch (error) {
      console.error('登录失败：', error)
      return false
    }
  }

  // 退出登录
  const logout = async () => {
    try {
      await axios.post('/api/auth/logout')
    } catch (error) {
      console.error('退出登录失败：', error)
    } finally {
      // 清除用户状态
      userInfo.value = {
        fuid: '',
        nickname: '',
        avatar: '',
        vipLevel: 0,
        vipExp: 0,
        signature: '',
        online: false
      }
      isLogin.value = false
      localStorage.removeItem('token')
    }
  }

  // 获取当前用户信息
  const fetchUserInfo = async () => {
    try {
      const res = await axios.get('/api/user/info')
      if (res.data.code === 200) {
        userInfo.value = res.data.data
        isLogin.value = true
      }
    } catch (error) {
      console.error('获取用户信息失败：', error)
      isLogin.value = false
    }
  }

  // 更新用户资料
  const updateUserProfile = async (profileData) => {
    try {
      const res = await axios.put('/api/user/profile', profileData)
      if (res.data.code === 200) {
        userInfo.value = { ...userInfo.value, ...profileData }
        return true
      }
      return false
    } catch (error) {
      console.error('更新用户资料失败：', error)
      return false
    }
  }

  // 切换主题
  const toggleTheme = () => {
    theme.value = theme.value === 'light' ? 'dark' : 'light'
    localStorage.setItem('theme', theme.value)
  }

  // 初始化主题（从本地存储读取）
  const initTheme = () => {
    const savedTheme = localStorage.getItem('theme')
    if (savedTheme) {
      theme.value = savedTheme
    }
  }

  return {
    userInfo,
    isLogin,
    theme,
    login,
    logout,
    fetchUserInfo,
    updateUserProfile,
    toggleTheme,
    initTheme
  }
})