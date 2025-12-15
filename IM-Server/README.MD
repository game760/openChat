### 一、项目介绍
本套代码是一个即时通讯（IM）系统的后端API服务，并未完成全部功能，有能力的可以二开继续开发，主要功能包括：
用户管理（注册、身份验证、信息存储等）
社交关系处理（好友关系管理）
群聊功能（群创建、成员管理、群公告等）
消息处理（单聊 / 群聊消息收发、离线消息存储、消息撤回等）
系统配置管理、数据存储、加密安全、限流控制等基础支撑功能

### 二、使用的技术栈
基础语言：Go
Web 框架：
Gin（HTTP 服务框架，处理 RESTful API）
Gorilla WebSocket /go-socket.io（实时通讯支持，处理 WebSocket 连接）
数据存储：
MySQL（关系型数据库，存储用户、好友、群聊、消息等核心数据，通过 GORM ORM 操作）
Redis（缓存 / 临时存储，用于限流、唯一 ID 校验、离线消息过期管理等）
MinIO（对象存储，可选，用于文件 / 图片存储）
安全加密：
bcrypt（密码加密存储）
AES-256（对称加密，用于数据加密）
RSA（非对称加密，用于密钥对加载与加解密）
JWT（用户身份认证，扩展包含设备信息）
配置与日志：
YAML（配置文件格式，通过gopkg.in/yaml.v3解析）
Logrus（日志框架）+ Lumberjack（日志文件轮转）
限流与安全：
令牌桶算法（golang.org/x/time/rate，用于 API 请求限流）
Cloudflare Turnstile（人机验证，防止恶意注册）
其他工具：
CORS 中间件（处理跨域请求）
TLS（生产环境 HTTPS 支持）

### 三、系统结构图
┌─────────────────────────────────────────────────────────────────┐  
│                      核心模块                                    │  
├─────────────┬─────────────┬─────────────┬───────────────────────┤  
│  配置模块    │  日志模块   │  存储模块   │      通信模块              │  
│ (loadConfig)│ (initLog)   │             │                     │  
└──────┬──────┴──────┬──────┴──────┬──────┴──────────┬──────────┘  
       │             │             │                 │  
┌──────▼──────┐┌──────▼──────┐┌──────▼──────┐┌────────▼──────────┐  
│ MySQL连接   ││ Redis连接   ││ MinIO连接   ││ HTTP服务 (Gin)    │  
│ (initMySQL) ││ (initRedis) ││ (initMinIO) ││ - 路由/中间件     │  
└──────┬──────┘└──────┬──────┘└──────┬──────┘│ - API接口         │  
       │             │             │      └────────┬──────────┘  
┌──────▼──────┐┌──────▼──────┐┌──────▼──────┐       │  
│ 数据模型    ││ 缓存/限流   ││ 文件存储    │       │  
│ (User/Friend││ (令牌桶/    ││ (本地/MinIO)│       │  
│  Group等)   ││  sync.Map)  ││             │       │  
└──────┬──────┘└─────────────┘└─────────────┘       │  
       │                                            │  
┌──────▼──────┐                           ┌────────▼──────────┐  
│ 业务逻辑    │                           │ WebSocket服务       │  
│ - 用户管理  │                           │ (socket.io)         │  
│ - 好友管理  │◄──────────────────────────┤ - 实时消息推送      │  
│ - 群聊管理  │                           │ - 连接维护         │  
│ - 消息处理  │                           │                   │  
└─────────────┘                           └───────────────────┘  

### API实现以下功能
```
// 公开接口分组（无需登录即可访问）
// 路径前缀：/api/v1/public
publicGroup := r.Group("/api/v1/public")
{
    // 用户注册接口
    // 请求方法：POST
    // 路径：/api/v1/public/register
    // 限流中间件：register_login_ip（IP维度限流）、register_login_user（用户维度限流）
    // 功能：处理用户注册逻辑，包括参数校验、人机验证、账号创建等
    publicGroup.POST("/register", limiters["register_login_ip"], limiters["register_login_user"], registerHandler)
    
    // 用户登录接口
    // 请求方法：POST
    // 路径：/api/v1/public/login
    // 限流中间件：register_login_ip（IP维度限流）、register_login_user（用户维度限流）
    // 功能：处理用户登录逻辑，验证账号密码并生成身份令牌（JWT）
    publicGroup.POST("/login", limiters["register_login_ip"], limiters["register_login_user"], loginHandler)
    
    // 服务健康检查接口
    // 请求方法：GET
    // 路径：/api/v1/public/health
    // 限流中间件：register_login_ip（IP维度限流）
    // 功能：返回服务运行状态（如启动时间、运行时长等），用于监测服务可用性
    publicGroup.GET("/health", limiters["register_login_ip"], healthHandler)
}

// 私有接口分组（需登录验证，通过authMiddleware()中间件校验JWT令牌）
// 路径前缀：/api/v1/private
privateGroup := r.Group("/api/v1/private")
privateGroup.Use(authMiddleware()) // 统一身份验证中间件，验证请求中的JWT令牌有效性
{
    // 设备管理相关接口
    // 获取当前用户的所有登录设备
    privateGroup.GET("/devices", getDevicesHandler)
    // 强制登出指定设备（踢下线）
    privateGroup.POST("/devices/kick", kickDeviceHandler)

    // 好友相关接口
    // 搜索好友（通过用户名、昵称或邮箱）
    privateGroup.GET("/friend/search", searchFriendHandler)
    // 发起添加好友请求
    privateGroup.POST("/friend/add", addFriendHandler)
    // 删除好友关系
    privateGroup.DELETE("/friend/:fuid", deleteFriendHandler)
    // 更新好友备注信息
    privateGroup.PUT("/friend/remark", updateFriendRemarkHandler)
    // 将指定用户加入黑名单
    privateGroup.POST("/friend/blacklist/add/:fuid", addBlacklistHandler)
    // 将指定用户从黑名单移除
    privateGroup.POST("/friend/blacklist/remove/:fuid", removeBlacklistHandler)
    // 获取指定好友的详细资料
    privateGroup.GET("/friend/profile/:fuid", getFriendProfileHandler)

    // 群聊相关接口
    // 创建新群聊
    privateGroup.POST("/group/create", createGroupHandler)
    // 搜索群聊（通过群名或群ID）
    privateGroup.GET("/group/search", searchGroupHandler)
    // 加入群聊（需邀请或验证）
    privateGroup.POST("/group/join", joinGroupHandler)
    // 退出指定群聊
    privateGroup.DELETE("/group/quit/:quid", quitGroupHandler)
    // 禁言群内指定成员
    privateGroup.POST("/group/mute", groupMuteHandler)
    // 将指定成员踢出群聊
    privateGroup.POST("/group/kick", kickGroupMemberHandler)
    // 发布群公告
    privateGroup.POST("/group/notice/publish", publishGroupNoticeHandler)
    // 解散指定群聊（仅群主可操作）
    privateGroup.DELETE("/group/dissolve/:quid", dissolveGroupHandler)
    // 转让群聊 ownership（群主权限转移）
    privateGroup.POST("/group/transfer", transferGroupHandler)
    // 获取指定群聊的详细信息（群资料、成员列表等）
    privateGroup.GET("/group/profile/:quid", getGroupProfileHandler)

    // 消息相关接口
    // 发送消息（单聊/群聊）
    // 限流中间件：message_conn（消息发送频率限流）、group_user（群内用户操作限流）
    privateGroup.POST("/message/send", limiters["message_conn"], limiters["group_user"], sendMessageHandler)
    // 撤回指定消息（需在有效期内）
    privateGroup.POST("/message/recall/:msg_id", recallMessageHandler)
    // 获取离线消息（用户上线后同步未接收的消息）
    privateGroup.GET("/message/offline", getOfflineMessageHandler)
    // 获取未读消息总数
    privateGroup.GET("/message/unread/count", getUnreadMessageCountHandler)
    
    // 语音消息发送接口
    privateGroup.POST("/message/voice", authMiddleware(), sendVoiceMessageHandler)
    // 初始化通话（语音/视频）
    privateGroup.POST("/call/init", authMiddleware(), initCallHandler)
    // 接受通话请求
    privateGroup.POST("/call/accept", authMiddleware(), acceptCallHandler)
    // 拒绝通话请求
    privateGroup.POST("/call/reject", authMiddleware(), rejectCallHandler)
    // 结束当前通话
    privateGroup.POST("/call/end", authMiddleware(), endCallHandler)

    // 文件上传接口（支持图片、普通文件等，受存储配置限制）
    privateGroup.POST("/upload", uploadFileHandler)
}
```

### 部署步骤
1. 拉取代码
bash
运行
git clone <项目仓库地址>
cd <项目目录>
2. 安装依赖
bash
运行
go mod tidy  # 自动下载项目依赖
3. 构建项目
Linux/macOS：
bash
运行
GOOS=linux GOARCH=amd64 go build -o chat-server main.go
Windows：
bash
运行
GOOS=windows GOARCH=amd64 go build -o chat-server.exe main.go
5. 启动服务
开发环境（HTTP+WS，不验证 TLS）：
bash
运行
./chat-server  # 直接运行，默认加载config.yaml
6. 生产环境（HTTPS+WSS，需验证 TLS 证书）：
bash
运行
生产环境（HTTPS+WSS，需验证 TLS 证书）