<template>
  <q-app>
    <!-- 主题切换 -->
    <q-theme-provider :theme="currentTheme">
      <!-- 主布局 -->
      <q-layout view="hHh lpR fFf">
        <!-- 左侧菜单 -->
        <q-drawer
          v-model="drawerOpen"
          side="left"
          bordered
          content-class="bg-surface"
          :width="280"
          :breakpoint="0"
          persistent
        >
          <!-- 头部：软件名称 + 消息数提示 -->
          <q-toolbar class="bg-primary text-white">
            <q-toolbar-title>{{ appName }}</q-toolbar-title>
            <q-badge
              color="red"
              :label="totalUnreadCount"
              class="q-mx-sm"
              v-if="totalUnreadCount > 0"
            />
          </q-toolbar>

          <!-- 中间：联系人列表 -->
          <q-scroll-area class="fit">
            <!-- 标签页 -->
            <q-tabs v-model="activeTab" class="bg-surface-1">
              <q-tab label="最近联系" name="recent">
                <template v-slot:badge>
                  <q-badge color="red" :label="recentUnreadCount" v-if="recentUnreadCount > 0" />
                </template>
              </q-tab>
              <q-tab label="好友列表" name="friends">
                <template v-slot:badge>
                  <q-badge color="red" :label="friendUnreadCount" v-if="friendUnreadCount > 0" />
                </template>
              </q-tab>
              <q-tab label="群聊列表" name="groups">
                <template v-slot:badge>
                  <q-badge color="red" :label="groupUnreadCount" v-if="groupUnreadCount > 0" />
                </template>
              </q-tab>
              <q-tab label="黑名单" name="blacklist" />
            </q-tabs>

            <!-- 最近联系面板 -->
            <q-tab-panel name="recent" class="q-pa-sm">
              <div
                v-for="item in recentList"
                :key="item.id"
                class="q-pa-md cursor-pointer rounded-borders hover:bg-surface-1"
                @click="selectChat(item)"
                :class="{ 'bg-primary/10': selectedChat?.id === item.id }"
              >
                <div class="flex items-center">
                  <q-avatar size="40px" class="mr-3">
                    <img :src="item.avatar || defaultAvatar" :alt="item.name" />
                    <q-badge
                      color="green"
                      size="8px"
                      class="absolute bottom-0 right-0"
                      v-if="item.online"
                    />
                  </q-avatar>
                  <div class="flex-1">
                    <div class="flex justify-between items-center">
                      <span 
                        class="font-medium" 
                        :class="{ 'text-red-600': item.vipLevel > 0 }"
                      >
                        {{ item.name }}
                        <q-icon name="star" size="14px" class="text-yellow-500 ml-1" v-if="item.vipLevel > 0" />
                      </span>
                      <span class="text-caption text-grey-500">{{ item.lastMsgTime }}</span>
                    </div>
                    <div class="text-sm text-grey-600 truncate">
                      {{ item.lastMsgContent }}
                      <q-badge
                        color="red"
                        :label="item.unreadCount"
                        class="ml-2"
                        v-if="item.unreadCount > 0"
                      />
                    </div>
                  </div>
                </div>
              </div>
            </q-tab-panel>

            <!-- 好友列表面板 -->
            <q-tab-panel name="friends" class="q-pa-sm">
              <div
                v-for="friend in friendList"
                :key="friend.fuid"
                class="q-pa-md cursor-pointer rounded-borders hover:bg-surface-1"
                @click="selectChat(friend)"
                :class="{ 'bg-primary/10': selectedChat?.fuid === friend.fuid }"
              >
                <div class="flex items-center">
                  <q-avatar size="40px" class="mr-3">
                    <img :src="friend.avatar || defaultAvatar" :alt="friend.nickname" />
                    <q-badge
                      color="green"
                      size="8px"
                      class="absolute bottom-0 right-0"
                      v-if="friend.online"
                    />
                  </q-avatar>
                  <div class="flex-1">
                    <div class="flex justify-between items-center">
                      <span 
                        class="font-medium" 
                        :class="{ 'text-red-600': friend.vipLevel > 0 }"
                      >
                        {{ friend.remark || friend.nickname }}
                        <q-icon name="star" size="14px" class="text-yellow-500 ml-1" v-if="friend.vipLevel > 0" />
                      </span>
                      <q-badge
                        color="red"
                        :label="friend.unreadCount"
                        v-if="friend.unreadCount > 0"
                      />
                    </div>
                    <div class="text-sm text-grey-600">{{ friend.signature || '暂无签名' }}</div>
                  </div>
                  <q-icon
                    name="more_vert"
                    class="text-grey-500"
                    @click.stop="openFriendMenu(friend)"
                  />
                </div>
              </div>
            </q-tab-panel>

            <!-- 群聊列表面板 -->
            <q-tab-panel name="groups" class="q-pa-sm">
              <div
                v-for="group in groupList"
                :key="group.quid"
                class="q-pa-md cursor-pointer rounded-borders hover:bg-surface-1"
                @click="selectChat(group)"
                :class="{ 'bg-primary/10': selectedChat?.quid === group.quid }"
              >
                <div class="flex items-center">
                  <q-avatar size="40px" class="mr-3">
                    <img :src="group.avatar || defaultGroupAvatar" :alt="group.name" />
                  </q-avatar>
                  <div class="flex-1">
                    <div class="flex justify-between items-center">
                      <span 
                        class="font-medium" 
                        :class="{ 'text-red-600': group.vipLevel > 0 }"
                      >
                        {{ group.name }}
                        <q-icon name="star" size="14px" class="text-yellow-500 ml-1" v-if="group.vipLevel > 0" />
                      </span>
                      <q-badge
                        color="red"
                        :label="group.unreadCount"
                        v-if="group.unreadCount > 0"
                      />
                    </div>
                    <div class="text-sm text-grey-600">
                      {{ group.memberCount }}人 | {{ group.desc || '暂无介绍' }}
                    </div>
                  </div>
                  <q-icon
                    name="more_vert"
                    class="text-grey-500"
                    @click.stop="openGroupMenu(group)"
                  />
                </div>
              </div>
            </q-tab-panel>

            <!-- 黑名单面板 -->
            <q-tab-panel name="blacklist" class="q-pa-sm">
              <div
                v-for="black in blacklist"
                :key="black.fuid"
                class="q-pa-md cursor-pointer rounded-borders hover:bg-surface-1"
              >
                <div class="flex items-center">
                  <q-avatar size="40px" class="mr-3">
                    <img :src="black.avatar || defaultAvatar" :alt="black.nickname" />
                  </q-avatar>
                  <div class="flex-1">
                    <div class="font-medium">{{ black.remark || black.nickname }}</div>
                    <div class="text-sm text-grey-600">{{ black.fuid }}</div>
                  </div>
                  <q-btn
                    size="sm"
                    label="移出"
                    color="primary"
                    @click="removeFromBlacklist(black)"
                  />
                </div>
              </div>
            </q-tab-panel>
          </q-scroll-area>

          <!-- 底部：个人设置、系统设置、退出登录 -->
          <q-separator />
          <div class="q-pa-md">
            <q-list>
              <q-item clickable @click="openUserProfile">
                <q-item-section avatar>
                  <q-avatar>
                    <img :src="userInfo.avatar || defaultAvatar" :alt="userInfo.nickname" />
                  </q-avatar>
                </q-item-section>
                <q-item-section>
                  <q-item-label>{{ userInfo.nickname }}</q-item-label>
                  <q-item-label caption>{{ userInfo.fuid }}</q-item-label>
                </q-item-section>
              </q-item>
              <q-item clickable @click="toggleTheme">
                <q-item-section avatar>
                  <q-icon :name="currentTheme === 'light' ? 'dark_mode' : 'light_mode'" />
                </q-item-section>
                <q-item-section>
                  <q-item-label>{{ currentTheme === 'light' ? '切换暗色主题' : '切换亮色主题' }}</q-item-label>
                </q-item-section>
              </q-item>
              <q-item clickable @click="openSystemSettings">
                <q-item-section avatar>
                  <q-icon name="settings" />
                </q-item-section>
                <q-item-section>
                  <q-item-label>系统设置</q-item-label>
                </q-item-section>
              </q-item>
              <q-item clickable color="red" @click="logout">
                <q-item-section avatar>
                  <q-icon name="logout" />
                </q-item-section>
                <q-item-section>
                  <q-item-label>退出登录</q-item-label>
                </q-item-section>
              </q-item>
            </q-list>
          </div>
        </q-drawer>

        <!-- 右侧聊天窗口 -->
        <q-page-container v-if="selectedChat">
          <!-- 聊天窗口头部 -->
          <q-toolbar class="bg-surface border-b">
            <q-toolbar-title>
              <div class="flex items-center">
                <q-avatar size="32px" class="mr-2">
                  <img 
                    :src="selectedChat.avatar || (selectedChat.quid ? defaultGroupAvatar : defaultAvatar)" 
                    :alt="selectedChat.name || selectedChat.nickname"
                  />
                </q-avatar>
                <span 
                  :class="{ 'text-red-600': selectedChat.vipLevel > 0 }"
                >
                  {{ selectedChat.name || selectedChat.nickname || selectedChat.remark }}
                </span>
              </div>
            </q-toolbar-title>
            <q-space />
            <q-btn flat icon="search" @click="searchChatHistory" />
            <q-btn flat icon="more_vert" @click="openChatInfo" />
          </q-toolbar>

          <!-- 聊天消息显示区域 -->
          <q-page class="chat-content q-pa-sm">
            <q-scroll-area class="fit">
              <div class="chat-messages">
                <!-- 系统消息 -->
                <div 
                  v-for="msg in chatMessages"
                  :key="msg.msg_id"
                  class="chat-message q-mb-4"
                >
                  <div v-if="msg.content_type === 5" class="text-center q-my-2">
                    <span class="text-xs bg-grey-200 px-3 py-1 rounded-full text-grey-600">
                      {{ msg.content }}
                    </span>
                  </div>

                  <!-- 他人消息 -->
                  <div 
                    v-else-if="msg.sender_fuid !== userInfo.fuid"
                    class="flex"
                  >
                    <q-avatar size="36px" class="mr-2 mt-1">
                      <img 
                        :src="getSenderAvatar(msg.sender_fuid)" 
                        :alt="getSenderName(msg.sender_fuid)"
                      />
                    </q-avatar>
                    <div class="flex-1">
                      <div class="text-xs text-grey-500 mb-1">{{ getSenderName(msg.sender_fuid) }}</div>
                      <div 
                        class="inline-block bg-white border rounded-lg p-2 max-w-[70%]"
                        :style="{ 
                          'font-family': msg.font_style || '思源黑体',
                          'font-size': msg.font_size + 'px',
                          'color': msg.font_color || '#000000'
                        }"
                      >
                        <!-- 文字消息 -->
                        <div v-if="msg.content_type === 1">
                          {{ decryptContent(msg.content) }}
                          <q-btn
                            size="xs"
                            icon="undo"
                            flat
                            class="text-grey-400"
                            v-if="msg.is_recalled"
                            disabled
                          >
                            消息已撤回
                          </q-btn>
                        </div>
                        <!-- 图片消息 -->
                        <div v-if="msg.content_type === 2">
                          <q-img
                            :src="decryptContent(msg.content)"
                            style="max-width: 200px; max-height: 200px"
                            class="rounded"
                            @click="previewImage(decryptContent(msg.content))"
                          />
                          <q-btn
                            size="xs"
                            icon="undo"
                            flat
                            class="text-grey-400"
                            v-if="msg.is_recalled"
                            disabled
                          >
                            消息已撤回
                          </q-btn>
                        </div>
                        <!-- 文件消息 -->
                        <div v-if="msg.content_type === 3" class="flex items-center">
                          <q-icon name="attach_file" class="text-primary mr-2" />
                          <a 
                            :href="decryptContent(msg.content)" 
                            target="_blank"
                            class="text-primary"
                          >
                            {{ getFileName(msg.content) }}
                          </a>
                          <q-btn
                            size="xs"
                            icon="undo"
                            flat
                            class="text-grey-400"
                            v-if="msg.is_recalled"
                            disabled
                          >
                            消息已撤回
                          </q-btn>
                        </div>
                        <!-- 表情消息 -->
                        <div v-if="msg.content_type === 4" class="text-2xl">
                          {{ decryptContent(msg.content) }}
                          <q-btn
                            size="xs"
                            icon="undo"
                            flat
                            class="text-grey-400"
                            v-if="msg.is_recalled"
                            disabled
                          >
                            消息已撤回
                          </q-btn>
                        </div>
                      </div>
                      <div class="text-xs text-grey-400 mt-1">{{ formatTime(msg.send_time) }}</div>
                    </div>
                  </div>

                  <!-- 自己的消息 -->
                  <div 
                    v-else
                    class="flex justify-end"
                  >
                    <div class="flex-1 text-right">
                      <div class="text-xs text-grey-500 mb-1">我</div>
                      <div 
                        class="inline-block bg-primary text-white rounded-lg p-2 max-w-[70%]"
                        :style="{ 
                          'font-family': msg.font_style || '思源黑体',
                          'font-size': msg.font_size + 'px',
                          'color': msg.font_color || '#ffffff'
                        }"
                      >
                        <!-- 文字消息 -->
                        <div v-if="msg.content_type === 1">
                          {{ decryptContent(msg.content) }}
                          <q-btn
                            size="xs"
                            icon="undo"
                            flat
                            class="text-white/70"
                            v-if="!msg.is_recalled && canRecall(msg.send_time)"
                            @click="recallMessage(msg)"
                          >
                            撤回
                          </q-btn>
                          <q-btn
                            size="xs"
                            icon="undo"
                            flat
                            class="text-white/70"
                            v-if="msg.is_recalled"
                            disabled
                          >
                            已撤回
                          </q-btn>
                        </div>
                        <!-- 图片消息 -->
                        <div v-if="msg.content_type === 2">
                          <q-img
                            :src="decryptContent(msg.content)"
                            style="max-width: 200px; max-height: 200px"
                            class="rounded"
                            @click="previewImage(decryptContent(msg.content))"
                          />
                          <q-btn
                            size="xs"
                            icon="undo"
                            flat
                            class="text-white/70"
                            v-if="!msg.is_recalled && canRecall(msg.send_time)"
                            @click="recallMessage(msg)"
                          >
                            撤回
                          </q-btn>
                          <q-btn
                            size="xs"
                            icon="undo"
                            flat
                            class="text-white/70"
                            v-if="msg.is_recalled"
                            disabled
                          >
                            已撤回
                          </q-btn>
                        </div>
                        <!-- 文件消息 -->
                        <div v-if="msg.content_type === 3" class="flex items-center">
                          <q-icon name="attach_file" class="mr-2" />
                          <a 
                            :href="decryptContent(msg.content)" 
                            target="_blank"
                            class="text-white"
                          >
                            {{ getFileName(msg.content) }}
                          </a>
                          <q-btn
                            size="xs"
                            icon="undo"
                            flat
                            class="text-white/70"
                            v-if="!msg.is_recalled && canRecall(msg.send_time)"
                            @click="recallMessage(msg)"
                          >
                            撤回
                          </q-btn>
                          <q-btn
                            size="xs"
                            icon="undo"
                            flat
                            class="text-white/70"
                            v-if="msg.is_recalled"
                            disabled
                          >
                            已撤回
                          </q-btn>
                        </div>
                        <!-- 表情消息 -->
                        <div v-if="msg.content_type === 4" class="text-2xl">
                          {{ decryptContent(msg.content) }}
                          <q-btn
                            size="xs"
                            icon="undo"
                            flat
                            class="text-white/70"
                            v-if="!msg.is_recalled && canRecall(msg.send_time)"
                            @click="recallMessage(msg)"
                          >
                            撤回
                          </q-btn>
                          <q-btn
                            size="xs"
                            icon="undo"
                            flat
                            class="text-white/70"
                            v-if="msg.is_recalled"
                            disabled
                          >
                            已撤回
                          </q-btn>
                        </div>
                      </div>
                      <div class="text-xs text-grey-400 mt-1">{{ formatTime(msg.send_time) }}</div>
                    </div>
                    <q-avatar size="36px" class="ml-2 mt-1">
                      <img :src="userInfo.avatar || defaultAvatar" :alt="userInfo.nickname" />
                    </q-avatar>
                  </div>
                </div>
              </div>
            </q-scroll-area>
          </q-page>

          <!-- 聊天工具栏 -->
          <q-separator />
          <div class="chat-toolbar q-pa-sm bg-surface">
            <q-btn flat icon="format_size" @click="openFontSettings" />
            <q-btn flat icon="color_lens" @click="openColorPicker" />
            <q-btn flat icon="text_fields" @click="toggleFontSize" />
            <q-btn flat icon="emoji_emotions" @click="toggleEmojiPicker" />
            <q-btn flat icon="attach_file" @click="openFileUpload" />
            <q-btn flat icon="screenshot" @click="captureScreen" />
            <q-space />
            <q-btn flat icon="mic" />
          </div>

          <!-- 输入框区域 -->
          <q-separator />
          <div class="chat-input q-pa-sm bg-surface">
            <q-input
              v-model="messageContent"
              type="textarea"
              rows="3"
              placeholder="输入消息..."
              class="mb-2"
              :style="{ 
                'font-family': currentFontStyle,
                'font-size': currentFontSize + 'px',
                'color': currentFontColor
              }"
              @keydown.enter.exact="sendMessage"
              @keydown.enter.shift="() => messageContent += '\n'"
            />
            <div class="flex justify-end">
              <q-btn
                label="发送"
                color="primary"
                @click="sendMessage"
                :disabled="!messageContent.trim()"
              />
            </div>
          </div>

          <!-- 版权信息 -->
          <q-footer class="text-center text-xs text-grey-500 q-py-sm border-t">
            © {{ new Date().getFullYear() }} {{ appName }} - 版权所有
          </q-footer>
        </q-page-container>

        <!-- 未选择聊天时的占位 -->
        <q-page-container v-else>
          <q-page class="flex flex-center">
            <div class="text-center">
              <q-icon name="chat" size="64px" class="text-grey-400 mb-4" />
              <h3 class="text-grey-600">请选择一个聊天</h3>
              <p class="text-grey-400">选择好友或群聊开始聊天</p>
            </div>
          </q-page>
        </q-page-container>
      </q-layout>

      <!-- 右下角通知组件 -->
      <q-notification
        v-model="notifications"
        position="bottom-right"
        timeout="5000"
      />

      <!-- 表情选择器弹窗 -->
      <q-dialog v-model="emojiPickerOpen">
        <q-card class="emoji-picker">
          <q-card-header>
            <q-card-title>选择表情</q-card-title>
          </q-card-header>
          <q-card-section>
            <div class="grid grid-cols-8 gap-2">
              <div 
                v-for="emoji in emojiList"
                :key="emoji"
                class="text-2xl text-center cursor-pointer hover:bg-grey-100 rounded"
                @click="selectEmoji(emoji)"
              >
                {{ emoji }}
              </div>
            </div>
          </q-card-section>
          <q-card-actions align="right">
            <q-btn label="关闭" flat @click="emojiPickerOpen = false" />
          </q-card-actions>
        </q-card>
      </q-dialog>

      <!-- 字体设置弹窗 -->
      <q-dialog v-model="fontSettingsOpen">
        <q-card>
          <q-card-header>
            <q-card-title>字体设置</q-card-title>
          </q-card-header>
          <q-card-section>
            <q-select
              v-model="currentFontStyle"
              label="字体样式"
              :options="fontStyles"
              class="mb-2"
            />
            <q-slider
              v-model="currentFontSize"
              label="字体大小"
              :min="12"
              :max="24"
              :step="1"
              class="mb-2"
            />
            <q-color
              v-model="currentFontColor"
              label="字体颜色"
              class="mb-2"
            />
          </q-card-section>
          <q-card-actions align="right">
            <q-btn label="取消" flat @click="fontSettingsOpen = false" />
            <q-btn label="确认" color="primary" @click="confirmFontSettings" />
          </q-card-actions>
        </q-card>
      </q-dialog>

      <!-- 图片预览弹窗 -->
      <q-dialog v-model="imagePreviewOpen">
        <q-card class="image-preview">
          <q-img
            :src="previewImageUrl"
            class="max-w-[90vw] max-h-[90vh]"
          />
          <q-card-actions align="right">
            <q-btn label="关闭" flat @click="imagePreviewOpen = false" />
          </q-card-actions>
        </q-card>
      </q-dialog>

      <!-- 好友资料卡弹窗 -->
      <q-dialog v-model="friendProfileOpen">
        <q-card style="width: 400px">
          <q-card-header>
            <q-card-title>好友资料</q-card-title>
          </q-card-header>
          <q-card-section>
            <div class="text-center mb-4">
              <q-avatar size="80px">
                <img :src="friendProfile.avatar || defaultAvatar" :alt="friendProfile.nickname" />
              </q-avatar>
              <h3 class="mt-2" :class="{ 'text-red-600': friendProfile.vipLevel > 0 }">
                {{ friendProfile.nickname }}
                <q-badge color="red" :label="friendProfile.vipLevel" class="ml-2">VIP</q-badge>
              </h3>
              <p class="text-sm text-grey-600">{{ friendProfile.fuid }}</p>
            </div>
            <q-list>
              <q-item>
                <q-item-section label>VIP经验</q-item-section>
                <q-item-section>{{ friendProfile.vipExp }}</q-item-section>
              </q-item>
              <q-item>
                <q-item-section label>个人说明</q-item-section>
                <q-item-section>{{ friendProfile.signature || '暂无' }}</q-item-section>
              </q-item>
            </q-list>
          </q-card-section>
          <q-card-actions align="right">
            <q-btn label="关闭" flat @click="friendProfileOpen = false" />
          </q-card-actions>
        </q-card>
      </q-dialog>

      <!-- 群聊资料卡弹窗 -->
      <q-dialog v-model="groupProfileOpen">
        <q-card style="width: 400px">
          <q-card-header>
            <q-card-title>群聊资料</q-card-title>
          </q-card-header>
          <q-card-section>
            <div class="text-center mb-4">
              <q-avatar size="80px">
                <img :src="groupProfile.avatar || defaultGroupAvatar" :alt="groupProfile.name" />
              </q-avatar>
              <h3 class="mt-2" :class="{ 'text-red-600': groupProfile.vipLevel > 0 }">
                {{ groupProfile.name }}
                <q-badge color="red" :label="groupProfile.vipLevel" class="ml-2">VIP</q-badge>
              </h3>
              <p class="text-sm text-grey-600">{{ groupProfile.quid }}</p>
            </div>
            <q-list>
              <q-item>
                <q-item-section label>群主</q-item-section>
                <q-item-section>{{ groupProfile.ownerNickname }} ({{ groupProfile.ownerFUID }})</q-item-section>
              </q-item>
              <q-item>
                <q-item-section label>群等级</q-item-section>
                <q-item-section>{{ groupProfile.vipLevel }}</q-item-section>
              </q-item>
              <q-item>
                <q-item-section label>群经验</q-item-section>
                <q-item-section>{{ groupProfile.vipExp }}</q-item-section>
              </q-item>
              <q-item>
                <q-item-section label>群说明</q-item-section>
                <q-item-section>{{ groupProfile.desc || '暂无' }}</q-item-section>
              </q-item>
            </q-list>
          </q-card-section>
          <q-card-actions align="right">
            <q-btn label="关闭" flat @click="groupProfileOpen = false" />
          </q-card-actions>
        </q-card>
      </q-dialog>

      <!-- 搜索聊天历史弹窗 -->
      <q-dialog v-model="searchHistoryOpen">
        <q-card style="width: 500px">
          <q-card-header>
            <q-card-title>搜索聊天记录</q-card-title>
          </q-card-header>
          <q-card-section>
            <q-input
              v-model="searchKeyword"
              label="输入搜索关键词"
              placeholder="输入关键词搜索"
              class="mb-4"
              @keyup.enter="doSearchHistory"
            />
            <q-btn 
              label="搜索" 
              color="primary" 
              class="mb-4"
              @click="doSearchHistory"
            />
            
            <div v-if="searchResults.length > 0">
              <q-list>
                <q-item
                  v-for="result in searchResults"
                  :key="result.msg_id"
                  class="q-pa-md"
                  @click="jumpToMessage(result)"
                >
                  <div class="text-sm">
                    <span class="font-medium">{{ getSenderName(result.sender_fuid) }}：</span>
                    <span v-html="highlightKeyword(decryptContent(result.content))"></span>
                  </div>
                  <div class="text-xs text-grey-500 mt-1">{{ formatTime(result.send_time) }}</div>
                </q-item>
              </q-list>
            </div>
            <div v-else-if="searchKeyword && searchResults.length === 0" class="text-center text-grey-500">
              未找到相关记录
            </div>
          </q-card-section>
          <q-card-actions align="right">
            <q-btn label="关闭" flat @click="searchHistoryOpen = false" />
          </q-card-actions>
        </q-card>
      </q-dialog>

      <!-- 好友操作菜单 -->
      <q-menu
        v-model="friendMenuOpen"
        :anchor="friendMenuAnchor"
        anchorClickEvent="click"
      >
        <q-list>
          <q-item clickable @click="viewFriendProfile">
            <q-item-section>查看资料</q-item-section>
          </q-item>
          <q-item clickable @click="sendFriendMessage">
            <q-item-section>发消息</q-item-section>
          </q-item>
          <q-item clickable @click="addToBlacklist(currentFriend)">
            <q-item-section>加入黑名单</q-item-section>
          </q-item>
        </q-list>
      </q-menu>

      <!-- 群聊操作菜单 -->
      <q-menu
        v-model="groupMenuOpen"
        :anchor="groupMenuAnchor"
        anchorClickEvent="click"
      >
        <q-list>
          <q-item clickable @click="viewGroupProfile">
            <q-item-section>查看群资料</q-item-section>
          </q-item>
          <q-item clickable @click="exitGroup">
            <q-item-section>退出群聊</q-item-section>
          </q-item>
        </q-list>
      </q-menu>

      <!-- 颜色选择器弹窗 -->
      <q-dialog v-model="colorPickerOpen">
        <q-card>
          <q-card-header>
            <q-card-title>选择文字颜色</q-card-title>
          </q-card-header>
          <q-card-section>
            <q-color
              v-model="tempFontColor"
              label="字体颜色"
              class="mb-2"
            />
            <div class="grid grid-cols-8 gap-2 mt-4">
              <div 
                v-for="color in presetColors"
                :key="color"
                :style="{ backgroundColor: color, width: '30px', height: '30px', borderRadius: '50%', cursor: 'pointer' }"
                @click="tempFontColor = color"
              ></div>
            </div>
          </q-card-section>
          <q-card-actions align="right">
            <q-btn label="取消" flat @click="colorPickerOpen = false" />
            <q-btn label="确认" color="primary" @click="confirmColorSelection" />
          </q-card-actions>
        </q-card>
      </q-dialog>
    </q-theme-provider>
  </q-app>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useQuasar } from 'quasar'
import { useUserStore } from '@/stores/user'
import { useMessageStore } from '@/stores/message'
import axios from 'axios'
import { io } from 'socket.io-client'
import { JSEncrypt } from 'jsencrypt'
import Twemoji from 'twemoji'
import { useRouter } from 'vue-router'

// 环境变量
const appName = import.meta.env.VITE_APP_NAME
const apiUrl = import.meta.env.VITE_API_URL
const cfSiteKey = import.meta.env.VITE_CF_SITE_KEY
const rsaPublicKey = import.meta.env.VITE_RSA_PUBLIC_KEY

// 全局实例
const $q = useQuasar()
const userStore = useUserStore()
const messageStore = useMessageStore()
const router = useRouter()

// 基础数据
const drawerOpen = ref(true)
const activeTab = ref('recent')
const selectedChat = ref(null)
const userInfo = ref({
  fuid: '',
  nickname: '',
  avatar: '',
  vipLevel: 0,
  vipExp: 0
})
const defaultAvatar = 'https://cdn.quasar.dev/img/avatar.png'
const defaultGroupAvatar = 'https://cdn.quasar.dev/img/avatar_group.png'

// 消息相关
const chatMessages = ref([])
const messageContent = ref('')
const notifications = ref([])

// 未读消息计数
const totalUnreadCount = ref(0)
const recentUnreadCount = ref(0)
const friendUnreadCount = ref(0)
const groupUnreadCount = ref(0)

// 列表数据
const recentList = ref([])
const friendList = ref([])
const groupList = ref([])
const blacklist = ref([])

// 字体设置
const fontStyles = ref([
  { label: '思源黑体', value: '思源黑体' },
  { label: '思源宋体', value: '思源宋体' },
  { label: '思源柔体', value: '思源柔体' }
])
const currentFontStyle = ref('思源黑体')
const currentFontSize = ref(14)
const currentFontColor = ref('#000000')
const fontSettingsOpen = ref(false)
const tempFontStyle = ref('')
const tempFontSize = ref(14)
const tempFontColor = ref('#000000')

// 表情选择器
const emojiPickerOpen = ref(false)
const emojiList = ref([
  '😀', '😃', '😄', '😁', '😆', '😅', '😂', '🤣',
  '😊', '😇', '🙂', '🙃', '😉', '😌', '😍', '🥰',
  '😘', '😗', '😙', '😚', '😋', '😛', '😜', '😝'
])

// 图片预览
const imagePreviewOpen = ref(false)
const previewImageUrl = ref('')

// 资料卡
const friendProfileOpen = ref(false)
const friendProfile = ref({})
const groupProfileOpen = ref(false)
const groupProfile = ref({})

// 主题切换
const currentTheme = ref('light')

// 搜索相关
const searchHistoryOpen = ref(false)
const searchKeyword = ref('')
const searchResults = ref([])

// 菜单相关
const friendMenuOpen = ref(false)
const friendMenuAnchor = ref(null)
const currentFriend = ref(null)
const groupMenuOpen = ref(false)
const groupMenuAnchor = ref(null)
const currentGroup = ref(null)

// 颜色选择器
const colorPickerOpen = ref(false)
const presetColors = ref([
  '#000000', '#FFFFFF', '#FF0000', '#00FF00', '#0000FF',
  '#FFFF00', '#FF00FF', '#00FFFF', '#800000', '#008000',
  '#000080', '#808000', '#800080', '#008080', '#C0C0C0'
])

// Socket.IO连接
let socket = null

// 初始化RSA加密
const rsaEncrypt = (content) => {
  const encrypt = new JSEncrypt()
  encrypt.setPublicKey(rsaPublicKey)
  return encrypt.encrypt(content)
}

// RSA解密（前端仅用于展示，实际解密在后端）
const decryptContent = (content) => {
  try {
    // 前端仅演示，实际项目中前端不存储私钥
    return content
  } catch (e) {
    return '解密失败'
  }
}

// 格式化时间
const formatTime = (timeStr) => {
  const date = new Date(timeStr)
  return date.toLocaleString()
}

// 检查是否可以撤回消息（3分钟内）
const canRecall = (sendTime) => {
  const sendDate = new Date(sendTime)
  const now = new Date()
  const diff = (now - sendDate) / 1000
  return diff < 180
}

// 获取发送者信息
const getSenderAvatar = (fuid) => {
  const friend = friendList.value.find(item => item.fuid === fuid)
  return friend ? friend.avatar : defaultAvatar
}

const getSenderName = (fuid) => {
  const friend = friendList.value.find(item => item.fuid === fuid)
  return friend ? (friend.remark || friend.nickname) : '未知用户'
}

// 获取文件名
const getFileName = (content) => {
  const url = decryptContent(content)
  return url.substring(url.lastIndexOf('/') + 1)
}

// 主题切换
const toggleTheme = () => {
  currentTheme.value = currentTheme.value === 'light' ? 'dark' : 'light'
  $q.dark.set(currentTheme.value === 'dark')
  localStorage.setItem('theme', currentTheme.value)
}

// 选择聊天
const selectChat = (chatItem) => {
  selectedChat.value = chatItem
  // 加载聊天记录
  loadChatHistory(chatItem)
  // 标记已读
  markAsRead(chatItem)
}

// 加载聊天记录
const loadChatHistory = async (chatItem) => {
  try {
    const params = {
      receiverType: chatItem.quid ? 2 : 1,
      receiverId: chatItem.quid || chatItem.fuid
    }
    const res = await axios.get(`${apiUrl}/private/message/history`, { params })
    chatMessages.value = res.data.data || []
    // 滚动到底部
    scrollToBottom()
  } catch (e) {
    $q.notify({
      type: 'negative',
      message: '加载聊天记录失败'
    })
  }
}

// 标记消息已读
const markAsRead = async (chatItem) => {
  try {
    await axios.post(`${apiUrl}/private/message/read`, {
      receiverType: chatItem.quid ? 2 : 1,
      receiverId: chatItem.quid || chatItem.fuid
    })
    // 更新未读计数
    updateUnreadCount()
  } catch (e) {
    console.error('标记已读失败', e)
  }
}

// 发送消息
const sendMessage = async () => {
  if (!messageContent.value.trim() || !selectedChat.value) return

  try {
    // RSA加密消息内容
    const encryptedContent = rsaEncrypt(messageContent.value)
    
    const reqData = {
      receiver_type: selectedChat.value.quid ? 2 : 1,
      receiver_id: selectedChat.value.quid || selectedChat.value.fuid,
      content_type: 1, // 文字消息
      content: encryptedContent,
      font_style: currentFontStyle.value,
      font_size: currentFontSize.value,
      font_color: currentFontColor.value
    }

    const res = await axios.post(`${apiUrl}/private/message/send`, reqData)
    
    // 添加到聊天记录
    const newMsg = {
      msg_id: res.data.data.msg_id,
      sender_fuid: userInfo.value.fuid,
      receiver_type: reqData.receiver_type,
      receiver_id: reqData.receiver_id,
      content_type: 1,
      content: encryptedContent,
      font_style: currentFontStyle.value,
      font_size: currentFontSize.value,
      font_color: currentFontColor.value,
      is_recalled: false,
      send_time: new Date().toISOString()
    }
    chatMessages.value.push(newMsg)
    messageContent.value = ''

    // 滚动到底部
    scrollToBottom()
  } catch (e) {
    $q.notify({
      type: 'negative',
      message: '发送消息失败'
    })
    console.error('发送消息失败', e)
  }
}

// 撤回消息
const recallMessage = async (msg) => {
  try {
    await axios.post(`${apiUrl}/private/message/recall/${msg.msg_id}`)
    // 更新本地消息状态
    const index = chatMessages.value.findIndex(item => item.msg_id === msg.msg_id)
    if (index !== -1) {
      chatMessages.value[index].is_recalled = true
    }
    $q.notify({
      type: 'positive',
      message: '消息已撤回'
    })
  } catch (e) {
    $q.notify({
      type: 'negative',
      message: '撤回消息失败'
    })
    console.error('撤回消息失败', e)
  }
}

// 选择表情
const selectEmoji = (emoji) => {
  messageContent.value += emoji
  emojiPickerOpen.value = false
}

// 打开字体设置
const openFontSettings = () => {
  // 保存当前设置作为临时值
  tempFontStyle.value = currentFontStyle.value
  tempFontSize.value = currentFontSize.value
  tempFontColor.value = currentFontColor.value
  fontSettingsOpen.value = true
}

// 确认字体设置
const confirmFontSettings = () => {
  currentFontStyle.value = tempFontStyle.value
  currentFontSize.value = tempFontSize.value
  currentFontColor.value = tempFontColor.value
  fontSettingsOpen.value = false
}

// 切换字体大小
const toggleFontSize = () => {
  if (currentFontSize.value < 24) {
    currentFontSize.value += 2
  } else {
    currentFontSize.value = 12
  }
}

// 打开颜色选择器
const openColorPicker = () => {
  tempFontColor.value = currentFontColor.value
  colorPickerOpen.value = true
}

// 确认颜色选择
const confirmColorSelection = () => {
  currentFontColor.value = tempFontColor.value
  colorPickerOpen.value = false
}

// 打开文件上传
const openFileUpload = () => {
  if (!selectedChat.value) {
    $q.notify({
      type: 'warning',
      message: '请先选择聊天对象'
    })
    return
  }
  
  // 创建隐藏的文件输入
  const fileInput = document.createElement('input')
  fileInput.type = 'file'
  fileInput.multiple = false
  
  fileInput.onchange = async (e) => {
    const file = e.target.files[0]
    if (!file) return
    
    try {
      $q.loading.show({ message: '上传中...' })
      
      const formData = new FormData()
      formData.append('file', file)
      formData.append('receiver_type', selectedChat.value.quid ? 2 : 1)
      formData.append('receiver_id', selectedChat.value.quid || selectedChat.value.fuid)
      
      const res = await axios.post(`${apiUrl}/private/file/upload`, formData, {
        headers: { 'Content-Type': 'multipart/form-data' }
      })
      
      // 根据文件类型判断消息类型
      const contentType = file.type.startsWith('image/') ? 2 : 3
      
      // 添加到聊天记录
      const newMsg = {
        msg_id: res.data.data.msg_id,
        sender_fuid: userInfo.value.fuid,
        receiver_type: selectedChat.value.quid ? 2 : 1,
        receiver_id: selectedChat.value.quid || selectedChat.value.fuid,
        content_type: contentType,
        content: rsaEncrypt(res.data.data.url),
        is_recalled: false,
        send_time: new Date().toISOString()
      }
      chatMessages.value.push(newMsg)
      
      $q.notify({
        type: 'positive',
        message: '文件上传成功'
      })
      
      // 滚动到底部
      scrollToBottom()
    } catch (e) {
      $q.notify({
        type: 'negative',
        message: '文件上传失败'
      })
      console.error('文件上传失败', e)
    } finally {
      $q.loading.hide()
    }
  }
  
  fileInput.click()
}

// 截图功能
const captureScreen = async () => {
  if (!selectedChat.value) {
    $q.notify({
      type: 'warning',
      message: '请先选择聊天对象'
    })
    return
  }
  
  try {
    // 检查是否有权限
    const stream = await navigator.mediaDevices.getDisplayMedia({
      video: { mediaSource: 'screen' }
    })
    
    // 创建视频元素播放流
    const video = document.createElement('video')
    video.srcObject = stream
    await video.play()
    
    // 创建canvas捕获一帧
    const canvas = document.createElement('canvas')
    canvas.width = video.videoWidth
    canvas.height = video.videoHeight
    const ctx = canvas.getContext('2d')
    ctx.drawImage(video, 0, 0, canvas.width, canvas.height)
    
    // 停止流
    stream.getTracks().forEach(track => track.stop())
    
    // 转换为图片
    canvas.toBlob(async (blob) => {
      try {
        $q.loading.show({ message: '上传截图...' })
        
        const formData = new FormData()
        formData.append('file', blob, 'screenshot.png')
        formData.append('receiver_type', selectedChat.value.quid ? 2 : 1)
        formData.append('receiver_id', selectedChat.value.quid || selectedChat.value.fuid)
        
        const res = await axios.post(`${apiUrl}/private/file/upload`, formData, {
          headers: { 'Content-Type': 'multipart/form-data' }
        })
        
        // 添加到聊天记录
        const newMsg = {
          msg_id: res.data.data.msg_id,
          sender_fuid: userInfo.value.fuid,
          receiver_type: selectedChat.value.quid ? 2 : 1,
          receiver_id: selectedChat.value.quid || selectedChat.value.fuid,
          content_type: 2, // 图片消息
          content: rsaEncrypt(res.data.data.url),
          is_recalled: false,
          send_time: new Date().toISOString()
        }
        chatMessages.value.push(newMsg)
        
        $q.notify({
          type: 'positive',
          message: '截图发送成功'
        })
        
        // 滚动到底部
        scrollToBottom()
      } catch (e) {
        $q.notify({
          type: 'negative',
          message: '截图上传失败'
        })
        console.error('截图上传失败', e)
      } finally {
        $q.loading.hide()
      }
    })
  } catch (e) {
    $q.notify({
      type: 'negative',
      message: '无法获取屏幕捕获权限'
    })
    console.error('截图失败', e)
  }
}

// 搜索聊天历史
const searchChatHistory = () => {
  if (!selectedChat.value) return
  searchHistoryOpen.value = true
  searchKeyword.value = ''
  searchResults.value = []
}

// 执行搜索
const doSearchHistory = async () => {
  if (!searchKeyword.value.trim() || !selectedChat.value) return
  
  try {
    $q.loading.show({ message: '搜索中...' })
    
    const params = {
      receiverType: selectedChat.value.quid ? 2 : 1,
      receiverId: selectedChat.value.quid || selectedChat.value.fuid,
      keyword: searchKeyword.value
    }
    
    const res = await axios.get(`${apiUrl}/private/message/search`, { params })
    searchResults.value = res.data.data || []
  } catch (e) {
    $q.notify({
      type: 'negative',
      message: '搜索失败'
    })
    console.error('搜索失败', e)
  } finally {
    $q.loading.hide()
  }
}

// 高亮搜索关键词
const highlightKeyword = (content) => {
  if (!searchKeyword.value) return content
  const reg = new RegExp(`(${searchKeyword.value})`, 'gi')
  return content.replace(reg, '<span class="bg-yellow-200">$1</span>')
}

// 跳转到消息位置
const jumpToMessage = (msg) => {
  searchHistoryOpen.value = false
  
  nextTick(() => {
    const msgElement = document.querySelector(`[data-msg-id="${msg.msg_id}"]`)
    if (msgElement) {
      msgElement.scrollIntoView({ behavior: 'smooth', block: 'center' })
      // 添加高亮效果
      msgElement.classList.add('bg-primary/20')
      setTimeout(() => {
        msgElement.classList.remove('bg-primary/20')
      }, 2000)
    }
  })
}

// 打开聊天信息
const openChatInfo = () => {
  if (!selectedChat.value) return
  
  if (selectedChat.value.quid) {
    // 群聊
    fetchGroupProfile(selectedChat.value.quid)
  } else {
    // 好友
    fetchFriendProfile(selectedChat.value.fuid)
  }
}

// 获取好友资料
const fetchFriendProfile = async (fuid) => {
  try {
    const res = await axios.get(`${apiUrl}/user/profile/${fuid}`)
    friendProfile.value = res.data.data
    friendProfileOpen.value = true
  } catch (e) {
    $q.notify({
      type: 'negative',
      message: '获取好友资料失败'
    })
    console.error('获取好友资料失败', e)
  }
}

// 获取群聊资料
const fetchGroupProfile = async (quid) => {
  try {
    const res = await axios.get(`${apiUrl}/group/profile/${quid}`)
    groupProfile.value = res.data.data
    groupProfileOpen.value = true
  } catch (e) {
    $q.notify({
      type: 'negative',
      message: '获取群聊资料失败'
    })
    console.error('获取群聊资料失败', e)
  }
}

// 打开好友菜单
const openFriendMenu = (friend, e) => {
  currentFriend.value = friend
  friendMenuAnchor.value = e.target
  friendMenuOpen.value = true
}

// 打开群聊菜单
const openGroupMenu = (group, e) => {
  currentGroup.value = group
  groupMenuAnchor.value = e.target
  groupMenuOpen.value = true
}

// 查看好友资料
const viewFriendProfile = () => {
  friendMenuOpen.value = false
  fetchFriendProfile(currentFriend.value.fuid)
}

// 查看群聊资料
const viewGroupProfile = () => {
  groupMenuOpen.value = false
  fetchGroupProfile(currentGroup.value.quid)
}

// 发送好友消息
const sendFriendMessage = () => {
  friendMenuOpen.value = false
  selectChat(currentFriend.value)
}

// 加入黑名单
const addToBlacklist = async (friend) => {
  try {
    await axios.post(`${apiUrl}/user/blacklist/add`, { fuid: friend.fuid })
    $q.notify({
      type: 'positive',
      message: '已加入黑名单'
    })
    friendMenuOpen.value = false
    // 刷新列表
    loadFriendList()
    loadBlacklist()
  } catch (e) {
    $q.notify({
      type: 'negative',
      message: '操作失败'
    })
    console.error('加入黑名单失败', e)
  }
}

// 从黑名单移除
const removeFromBlacklist = async (black) => {
  try {
    await axios.post(`${apiUrl}/user/blacklist/remove`, { fuid: black.fuid })
    $q.notify({
      type: 'positive',
      message: '已移出黑名单'
    })
    // 刷新列表
    loadFriendList()
    loadBlacklist()
  } catch (e) {
    $q.notify({
      type: 'negative',
      message: '操作失败'
    })
    console.error('移出黑名单失败', e)
  }
}

// 退出群聊
const exitGroup = async () => {
  $q.dialog({
    title: '确认退出',
    message: `确定要退出 ${currentGroup.value.name} 吗？`,
    cancel: true,
    persistent: true
  }).onOk(async () => {
    try {
      await axios.post(`${apiUrl}/group/exit`, { quid: currentGroup.value.quid })
      $q.notify({
        type: 'positive',
        message: '已退出群聊'
      })
      groupMenuOpen.value = false
      // 如果当前正在聊天的是这个群，取消选择
      if (selectedChat.value && selectedChat.value.quid === currentGroup.value.quid) {
        selectedChat.value = null
      }
      // 刷新列表
      loadGroupList()
    } catch (e) {
      $q.notify({
        type: 'negative',
        message: '退出失败'
      })
      console.error('退出群聊失败', e)
    }
  })
}

// 打开用户个人资料
const openUserProfile = () => {
  router.push('/profile')
}

// 打开系统设置
const openSystemSettings = () => {
  router.push('/settings')
}

// 退出登录
const logout = () => {
  $q.dialog({
    title: '确认退出',
    message: '确定要退出登录吗？',
    cancel: true,
    persistent: true
  }).onOk(async () => {
    try {
      await axios.post(`${apiUrl}/auth/logout`)
      userStore.clearUserInfo()
      socket?.disconnect()
      router.push('/login')
    } catch (e) {
      console.error('退出登录失败', e)
      router.push('/login')
    }
  })
}

// 预览图片
const previewImage = (url) => {
  previewImageUrl.value = url
  imagePreviewOpen.value = true
}

// 切换表情选择器
const toggleEmojiPicker = () => {
  emojiPickerOpen.value = !emojiPickerOpen.value
}

// 滚动到聊天底部
const scrollToBottom = () => {
  setTimeout(() => {
    const scrollArea = document.querySelector('.q-scroll-area')
    if (scrollArea) {
      scrollArea.scrollTop = scrollArea.scrollHeight
    }
  }, 100)
}

// 更新未读消息计数
const updateUnreadCount = async () => {
  try {
    const res = await axios.get(`${apiUrl}/private/message/unread/count`)
    const data = res.data.data || {}
    totalUnreadCount.value = data.total || 0
    recentUnreadCount.value = data.recent || 0
    friendUnreadCount.value = data.friends || 0
    groupUnreadCount.value = data.groups || 0
  } catch (e) {
    console.error('获取未读计数失败', e)
  }
}

// 加载最近联系人
const loadRecentList = async () => {
  try {
    const res = await axios.get(`${apiUrl}/private/contact/recent`)
    recentList.value = res.data.data || []
  } catch (e) {
    console.error('加载最近联系人失败', e)
  }
}

// 加载好友列表
const loadFriendList = async () => {
  try {
    const res = await axios.get(`${apiUrl}/user/friends`)
    friendList.value = res.data.data || []
  } catch (e) {
    console.error('加载好友列表失败', e)
  }
}

// 加载群聊列表
const loadGroupList = async () => {
  try {
    const res = await axios.get(`${apiUrl}/group/list`)
    groupList.value = res.data.data || []
  } catch (e) {
    console.error('加载群聊列表失败', e)
  }
}

// 加载黑名单
const loadBlacklist = async () => {
  try {
    const res = await axios.get(`${apiUrl}/user/blacklist`)
    blacklist.value = res.data.data || []
  } catch (e) {
    console.error('加载黑名单失败', e)
  }
}

// 初始化Socket连接
const initSocket = () => {
  // 断开现有连接
  if (socket) {
    socket.disconnect()
  }
  
  // 创建新连接
  socket = io(apiUrl, {
    auth: {
      token: localStorage.getItem('token')
    }
  })
  
  // 连接成功
  socket.on('connect', () => {
    console.log('Socket connected')
  })
  
  // 接收新消息
  socket.on('new_message', (msg) => {
    // 如果是当前聊天的消息，直接添加到列表
    if (selectedChat.value) {
      const isCurrentChat = 
        (selectedChat.value.fuid && msg.sender_fuid === selectedChat.value.fuid) ||
        (selectedChat.value.quid && msg.receiver_id === selectedChat.value.quid)
      
      if (isCurrentChat) {
        chatMessages.value.push(msg)
        scrollToBottom()
        // 标记为已读
        markAsRead(selectedChat.value)
        return
      }
    }
    
    // 否则显示通知
    notifications.value.push({
      type: 'info',
      message: `${getSenderName(msg.sender_fuid)}: ${decryptContent(msg.content)}`,
      actions: [
        {
          label: '查看',
          handler: () => {
            // 找到对应的聊天对象
            const chatItem = recentList.value.find(
              item => item.fuid === msg.sender_fuid || item.quid === msg.receiver_id
            )
            if (chatItem) {
              selectChat(chatItem)
            }
          }
        }
      ]
    })
    
    // 更新未读计数和列表
    updateUnreadCount()
    loadRecentList()
  })
  
  // 消息撤回
  socket.on('message_recalled', (msgId) => {
    const index = chatMessages.value.findIndex(item => item.msg_id === msgId)
    if (index !== -1) {
      chatMessages.value[index].is_recalled = true
    }
  })
  
  // 连接错误
  socket.on('connect_error', (err) => {
    console.error('Socket connection error:', err)
  })
  
  // 断开连接
  socket.on('disconnect', (reason) => {
    console.log('Socket disconnected:', reason)
    // 如果不是手动断开，尝试重连
    if (reason !== 'io client disconnect') {
      socket.connect()
    }
  })
}

// 初始化
const init = async () => {
  // 获取用户信息
  const storedUser = userStore.getUserInfo()
  if (storedUser) {
    userInfo.value = storedUser
  } else {
    // 如果没有用户信息，跳转到登录页
    router.push('/login')
    return
  }
  
  // 加载主题设置
  const savedTheme = localStorage.getItem('theme')
  if (savedTheme) {
    currentTheme.value = savedTheme
    $q.dark.set(currentTheme.value === 'dark')
  }
  
  // 加载数据
  await Promise.all([
    loadRecentList(),
    loadFriendList(),
    loadGroupList(),
    loadBlacklist(),
    updateUnreadCount()
  ])
  
  // 初始化Socket
  initSocket()
}

// 组件挂载时
onMounted(() => {
  init()
})

// 组件卸载时
onUnmounted(() => {
  if (socket) {
    socket.disconnect()
  }
})
</script>

<style scoped>
.chat-content {
  height: calc(100vh - 260px);
  overflow: hidden;
}

.chat-toolbar {
  height: 50px;
}

.chat-input {
  height: 140px;
}

.chat-messages {
  padding-bottom: 20px;
}

.emoji-picker {
  max-width: 300px;
}

.image-preview {
  background-color: rgba(0, 0, 0, 0.9);
}

/* 消息高亮动画 */
@keyframes highlight {
  0% { background-color: transparent; }
  50% { background-color: rgba(66, 153, 225, 0.2); }
  100% { background-color: transparent; }
}

.highlight-animation {
  animation: highlight 2s ease-in-out;
}
</style>