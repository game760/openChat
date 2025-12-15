-- 用户表
CREATE TABLE `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `fuid` varchar(64) NOT NULL COMMENT '用户唯一标识',
  `username` varchar(64) NOT NULL COMMENT '用户名',
  `nickname` varchar(64) NOT NULL COMMENT '昵称',
  `email` varchar(128) NOT NULL COMMENT '邮箱',
  `password` varchar(128) NOT NULL COMMENT '密码(bcrypt加密)',
  `avatar` varchar(256) DEFAULT '' COMMENT '头像URL',
  `signature` varchar(256) DEFAULT '' COMMENT '个性签名',
  `vip_level` tinyint unsigned DEFAULT '0' COMMENT 'VIP等级',
  `vip_exp` bigint unsigned DEFAULT '0' COMMENT 'VIP经验值',
  `vip_start_time` datetime DEFAULT NULL COMMENT 'VIP开始时间',
  `status` tinyint unsigned DEFAULT '1' COMMENT '状态(1:正常 0:禁用)',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_fuid` (`fuid`),
  UNIQUE KEY `idx_username` (`username`),
  UNIQUE KEY `idx_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

-- 好友表
CREATE TABLE `friends` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `user_fuid` varchar(64) NOT NULL COMMENT '用户FUID',
  `friend_fuid` varchar(64) NOT NULL COMMENT '好友FUID',
  `remark` varchar(64) DEFAULT '' COMMENT '备注名',
  `status` tinyint unsigned DEFAULT '1' COMMENT '状态(1:正常 2:黑名单 0:已删除)',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_fuid` (`user_fuid`),
  KEY `idx_friend_fuid` (`friend_fuid`),
  KEY `idx_user_friend` (`user_fuid`,`friend_fuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='好友表';

-- 群聊表
CREATE TABLE `groups` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `quid` varchar(64) NOT NULL COMMENT '群唯一标识',
  `name` varchar(64) NOT NULL COMMENT '群名称',
  `owner_fuid` varchar(64) NOT NULL COMMENT '群主FUID',
  `avatar` varchar(256) DEFAULT '' COMMENT '群头像URL',
  `desc` varchar(256) DEFAULT '' COMMENT '群描述',
  `vip_level` tinyint unsigned DEFAULT '0' COMMENT '群VIP等级',
  `vip_exp` bigint unsigned DEFAULT '0' COMMENT '群VIP经验值',
  `vip_start_time` datetime DEFAULT NULL COMMENT '群VIP开始时间',
  `status` tinyint unsigned DEFAULT '1' COMMENT '状态(1:正常 0:解散)',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_quid` (`quid`),
  KEY `idx_owner_fuid` (`owner_fuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群聊表';

-- 群成员表
CREATE TABLE `group_members` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `group_quid` varchar(64) NOT NULL COMMENT '群QUID',
  `user_fuid` varchar(64) NOT NULL COMMENT '成员FUID',
  `role` tinyint unsigned DEFAULT '0' COMMENT '角色(0:普通 1:群主 2:管理)',
  `mute_end_time` datetime DEFAULT NULL COMMENT '禁言结束时间',
  `status` tinyint unsigned DEFAULT '1' COMMENT '状态(1:正常 0:已退出/踢出)',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_group_quid` (`group_quid`),
  KEY `idx_user_fuid` (`user_fuid`),
  KEY `idx_group_user` (`group_quid`,`user_fuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群成员表';

-- 消息表
CREATE TABLE `messages` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `msg_id` varchar(64) NOT NULL COMMENT '消息唯一ID',
  `sender_fuid` varchar(64) NOT NULL COMMENT '发送者FUID',
  `receiver_type` tinyint unsigned NOT NULL COMMENT '接收类型(1:单聊 2:群聊)',
  `receiver_id` varchar(64) NOT NULL COMMENT '接收者ID(单聊:好友FUID 群聊:群QUID)',
  `content_type` tinyint unsigned NOT NULL COMMENT '内容类型(1:文字 2:图片 3:文件 4:表情 5:系统消息)',
  `content` text NOT NULL COMMENT '加密后的内容',
  `font_style` varchar(64) DEFAULT '' COMMENT '字体样式',
  `font_size` int DEFAULT '14' COMMENT '字体大小',
  `font_color` varchar(16) DEFAULT '#000000' COMMENT '字体颜色',
  `is_recalled` tinyint(1) DEFAULT '0' COMMENT '是否撤回(0:否 1:是)',
  `is_read` tinyint(1) DEFAULT '0' COMMENT '是否已读(0:否 1:是)',
  `send_time` datetime NOT NULL COMMENT '发送时间',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_msg_id` (`msg_id`),
  KEY `idx_sender_fuid` (`sender_fuid`),
  KEY `idx_receiver` (`receiver_type`,`receiver_id`),
  KEY `idx_send_time` (`send_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='消息表';

-- 系统消息表
CREATE TABLE `system_messages` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `msg_id` varchar(64) NOT NULL COMMENT '消息唯一ID',
  `title` varchar(64) NOT NULL COMMENT '消息标题',
  `content` text NOT NULL COMMENT '消息内容',
  `target_type` tinyint unsigned NOT NULL COMMENT '目标类型(1:全体 2:指定用户 3:指定群)',
  `target_ids` text COMMENT '目标ID列表(逗号分隔)', -- 移除 DEFAULT ''
  `send_count` int DEFAULT '0' COMMENT '已发送次数',
  `max_send_count` int DEFAULT '1' COMMENT '最大发送次数',
  `cycle_send` tinyint(1) DEFAULT '0' COMMENT '是否循环发送(0:否 1:是)',
  `fixed_time` varchar(8) DEFAULT '' COMMENT '定点发送时间',
  `status` tinyint unsigned DEFAULT '0' COMMENT '状态(0:待发送 1:发送中 2:已完成)',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_msg_id` (`msg_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统消息表';
-- 离线消息表
CREATE TABLE `offline_messages` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `user_fuid` varchar(64) NOT NULL COMMENT '用户FUID',
  `msg_id` varchar(64) NOT NULL COMMENT '消息ID',
  `status` tinyint unsigned DEFAULT '0' COMMENT '状态(0:未推送 1:已推送)',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_fuid` (`user_fuid`),
  KEY `idx_user_status` (`user_fuid`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='离线消息表';

-- 群公告表
CREATE TABLE `group_notices` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `group_quid` varchar(64) NOT NULL COMMENT '群QUID',
  `content` text NOT NULL COMMENT '公告内容',
  `publisher_fuid` varchar(64) NOT NULL COMMENT '发布者FUID',
  `publish_time` datetime NOT NULL COMMENT '发布时间',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_group_quid` (`group_quid`),
  KEY `idx_publish_time` (`publish_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群公告表';