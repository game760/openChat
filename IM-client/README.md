### 项目介绍
该项目是一个基于 Vue 3 和 Quasar 框架开发的即时通讯（IM）应用，支持个人聊天、群聊等核心功能，提供了完善的消息管理、联系人管理和界面个性化设置能力，整体交互流畅，适配多场景使用。
- 技术亮点
  - 实时通信能力
    采用 Socket.IO 实现实时消息推送，通过initSocket建立长连接，监听new_message事件接收实时消息，确保消息即时性，并处理断线重连逻辑，提升通信稳定性。
- 加密机制
  - 集成JSEncrypt实现 RSA 加密，发送消息时对内容进行加密（rsaEncrypt函数），保障消息传输安全性（前端仅负责加密，解密在后端完成）。
- 组件化与响应式设计
  - 基于 Quasar UI 框架构建，使用q-dialog、q-list、q-tab等组件实现模块化界面，支持明暗主题切换（toggleTheme），并通过本地存储（localStorage）保存主题设置，适配不同用户偏好。
- 状态管理与路由
   -使用 Pinia（useUserStore、useMessageStore）管理用户信息和消息状态，结合 Vue Router 实现页面跳转（如登录页、个人资料页），实现状态与路由的联动。
- 异步操作与错误处理
  - 统一使用async/await处理 API 请求（如加载聊天记录、发送消息），结合 Quasar 的$q.notify和$q.loading提供操作反馈，错误场景（如消息发送失败、权限不足）处理完善。
- 交互体验优化
  - 实现消息高亮动画、滚动到最新消息、搜索结果定位等细节功能，通过防抖、节流（如搜索输入）提升交互流畅度，支持表情选择、字体样式 / 颜色自定义等个性化功能。
- 实现的功能
  - 核心聊天功能
  - 支持单聊（好友）和群聊，可发送文字消息、图片（含截图发送）；
  - 消息撤回（3 分钟内可操作）、已读标记（markAsRead）、历史记录加载与搜索（doSearchHistory）。
  - 联系人管理
  - 好友列表、群聊列表、最近联系人、黑名单的加载与展示；
  - 好友 / 群聊资料查看（fetchFriendProfile、fetchGroupProfile）、加入 / 移出黑名单。
  - 用户系统
  - 登录 / 退出登录（login、logout）、用户信息获取与更新（fetchUserInfo、updateUserProfile）；
  - VIP 标识展示（通过vipLevel区分）。
  - 界面与设置
  - 主题切换（明暗模式）、字体样式 / 大小 / 颜色自定义；
  - 聊天记录搜索高亮、图片预览、消息通知提示。
- 其他功能
  - 未读消息计数（updateUnreadCount）、群聊退出（exitGroup）、动态加载聊天数据等。
  
### 即时通讯（IM）应用部署说明
本文档详细说明该 Vue 3 + Quasar IM 应用的部署流程，涵盖环境准备、构建打包、部署方式（本地 / 服务器）、配置调整等核心环节，适配开发 / 测试 / 生产多环境场景。
一、部署环境准备
1. 基础环境要求
依赖项	版本要求	说明
Node.js	≥ 16.0.0（推荐 18.x LTS）	运行 / 打包 Vue 项目核心环境
npm/yarn/pnpm	对应 Node 版本兼容版本	包管理工具（推荐 pnpm）
服务器环境	Nginx/Apache/Tomcat	静态资源部署（推荐 Nginx）
后端服务	已部署的 IM 后端 / Socket 服务	需提前确认后端接口地址
2. 环境检查
bash
运行
# 检查Node版本
node -v
# 检查npm版本
npm -v
# （可选）安装pnpm（推荐）
npm install -g pnpm
二、代码拉取与依赖安装
1. 拉取项目代码
bash
运行
# 从Git仓库拉取（示例）
git clone https://xxx/im-app.git
cd im-app
2. 安装项目依赖
bash
运行
# 使用npm
npm install
# 或使用pnpm（推荐，速度更快）
pnpm install
三、环境配置调整
1. 核心配置文件
项目根目录下的 .env/.env.prod/.env.dev 文件，需根据部署环境修改以下核心配置：
env
# 后端API基础地址
VITE_API_BASE_URL = "https://im-api.xxx.com"
# Socket.IO服务地址
VITE_SOCKET_URL = "wss://im-socket.xxx.com"
# 静态资源CDN地址（生产环境可选）
VITE_CDN_URL = "https://cdn.xxx.com/im-app"
# 环境标识（dev/test/prod）
VITE_ENV = "prod"
2. 其他配置（可选）
主题默认配置：修改 src/settings/theme.js 调整默认主题（明暗模式、字体等）；
权限相关：修改 src/utils/permission.js 调整路由权限、接口权限校验规则；
消息加密：若需调整 RSA 加密公钥，修改 src/utils/encrypt.js 中的公钥配置。
四、项目打包构建
1. 开发环境打包（测试用）
bash
运行
# Quasar框架打包命令（开发环境）
quasar build --mode dev
# 或原生Vue打包（若未用Quasar）
npm run build:dev
2. 生产环境打包（正式部署）
bash
运行
# Quasar框架打包（生产环境，推荐）
quasar build --mode prod
# 或原生Vue打包
npm run build:prod