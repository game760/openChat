package main

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"net/url"
	"gopkg.in/yaml.v3"
	"crypto/tls"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	redis "github.com/go-redis/redis/v8"
	"golang.org/x/time/rate"
	gorillaWs "github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	transportWs "github.com/googollee/go-socket.io/engineio/transport/websocket"
	"gopkg.in/natefinch/lumberjack.v2"
	"github.com/golang-jwt/jwt/v5"
)

// 全局配置结构体
type Config struct {
	App struct {
		Name    string `yaml:"name"`
		Mode    string `yaml:"mode"`
		Version string `yaml:"version"`
		Port    struct {
			DevHTTP int `yaml:"dev_http"`
			DevWS   int `yaml:"dev_ws"`
			ProdHTTPS int `yaml:"prod_https"`
			ProdWSS int `yaml:"prod_wss"`
		} `yaml:"port"`
		// 新增TLS证书配置
		TLS struct {
			CertPath string `yaml:"cert_path"`
			KeyPath  string `yaml:"key_path"`
		} `yaml:"tls"`
		// 跨域配置
		CORS struct {
			AllowOrigins     []string `yaml:"allow_origins"`
			AllowMethods     []string `yaml:"allow_methods"`
			AllowHeaders     []string `yaml:"allow_headers"`
			AllowCredentials bool     `yaml:"allow_credentials"`
			MaxAge           int      `yaml:"max_age"`
		} `yaml:"cors"`
	} `yaml:"app"`
	MySQL struct {
		Host         string `yaml:"host"`
		Port         int    `yaml:"port"`
		User         string `yaml:"user"`
		Passwd       string `yaml:"passwd"`
		Database     string `yaml:"database"`
		Charset      string `yaml:"charset"`
		MaxOpenConns int    `yaml:"max_open_conns"`
		MaxIdleConns int    `yaml:"max_idle_conns"`
		ConnMaxLifetime int `yaml:"conn_max_lifetime"`
	} `yaml:"mysql"`
	Redis struct {
		Host            string `yaml:"host"`
		Port            int    `yaml:"port"`
		Password        string `yaml:"password"`
		DB              int    `yaml:"db"`
		PoolSize        int    `yaml:"pool_size"`
		IdleTimeout     int    `yaml:"idle_timeout"`
		OfflineMsgExpire int   `yaml:"offline_msg_expire"`
		LimitCacheExpire int  `yaml:"limit_cache_expire"`
	} `yaml:"redis"`
	Storage struct {
		Type string `yaml:"type"`
		Local struct {
			Path   string `yaml:"path"`
			Domain string `yaml:"domain"`
		} `yaml:"local"`
		MinIO struct {
			Endpoint  string `yaml:"endpoint"`
			AccessKey string `yaml:"access_key"`
			SecretKey string `yaml:"secret_key"`
			Bucket    string `yaml:"bucket"`
			UseSSL    bool   `yaml:"use_ssl"`
			Domain    string `yaml:"domain"`
		} `yaml:"minio"`
		File struct {
			MaxSize   int      `yaml:"max_size"`
			AllowTypes []string `yaml:"allow_types"`
		} `yaml:"file"`
		Image struct {
			MaxSize   int      `yaml:"max_size"`
			AllowTypes []string `yaml:"allow_types"`
		} `yaml:"image"`
	} `yaml:"storage"`
	Crypto struct {
		JWT struct {
			Secret       string `yaml:"secret"`
			Expire       int    `yaml:"expire"`
			RefreshExpire int   `yaml:"refresh_expire"`
		} `yaml:"jwt"`
		RSA struct {
            PublicKeyPath  string `yaml:"public_key_path"`
            PrivateKeyPath string `yaml:"private_key_path"`
		} `yaml:"rsa"`
	} `yaml:"crypto"`
	CFTurnstile struct {
		SiteKey   string `yaml:"site_key"`
		SecretKey string `yaml:"secret_key"`
	} `yaml:"cf_turnstile"`
	RateLimit struct {
		Bucket struct {
			Capacity int `yaml:"capacity"`
			Rate     int `yaml:"rate"`
		} `yaml:"bucket"`
		RegisterLoginIP struct {
			Limit  int `yaml:"limit"`
			Period int `yaml:"period"`
		} `yaml:"register_login_ip"`
		RegisterLoginUser struct {
			Limit  int `yaml:"limit"`
			Period int `yaml:"period"`
		} `yaml:"register_login_user"`
		MessageConn struct {
			Limit  int `yaml:"limit"`
			Period int `yaml:"period"`
		} `yaml:"message_conn"`
		MessageConcurrent struct {
			Limit int `yaml:"limit"`
		} `yaml:"message_concurrent"`
		GroupUser struct {
			Limit  int `yaml:"limit"`
			Period int `yaml:"period"`
		} `yaml:"group_user"`
	} `yaml:"rate_limit"`
	Business struct {
		User struct {
			FUIDLen     int `yaml:"fuid_len"`
			PasswordCost int `yaml:"password_cost"`
			FriendMax   int `yaml:"friend_max"`
		} `yaml:"user"`
		Group struct {
			QUIDLen        int `yaml:"quid_len"`
			CreateMax      int `yaml:"create_max"`
			MemberMax      int `yaml:"member_max"`
			AdminMax       int `yaml:"admin_max"`
			MuteTimeDefault int `yaml:"mute_time_default"`
			AtAllLimitAdmin int `yaml:"at_all_limit_admin"`
			AtAllLimitOwner int `yaml:"at_all_limit_owner"`
		} `yaml:"group"`
		VIP struct {
			LevelMax      int `yaml:"level_max"`
			ExpPerHour    int `yaml:"exp_per_hour"`
			UpdateInterval int `yaml:"update_interval"`
		} `yaml:"vip"`
		GroupVIP struct {
			LevelMax      int `yaml:"level_max"`
			ExpPerHour    int `yaml:"exp_per_hour"`
			UpdateInterval int `yaml:"update_interval"`
		} `yaml:"group_vip"`
		Message struct {
			RecallTimeout int `yaml:"recall_timeout"`
			AutoClean struct {
				Enable bool `yaml:"enable"`
				Days   int  `yaml:"days"`
			} `yaml:"auto_clean"`
			SystemMsg struct {
				CycleSend bool   `yaml:"cycle_send"`
				FixedTime string `yaml:"fixed_time"`
				SendTimes int    `yaml:"send_times"`
			} `yaml:"system_msg"`
		} `yaml:"message"`
		Notify struct {
			Ntfy struct {
				URL    string `yaml:"url"`
				Topic  string `yaml:"topic"`
				Enable bool   `yaml:"enable"`
			} `yaml:"ntfy"`
		} `yaml:"notify"`
	} `yaml:"business"`
	Log struct {
		Path          string `yaml:"path"`
		FileNameFormat string `yaml:"file_name_format"`
		MaxSize       int    `yaml:"max_size"`
		MaxBackups    int    `yaml:"max_backups"`
		MaxAge        int    `yaml:"max_age"`
		Compress      bool   `yaml:"compress"`
		Level         string `yaml:"level"`
	} `yaml:"log"`
}

// 全局变量
var (
	startTime time.Time
	socketServer *socketio.Server
	cfg        Config
	db         *gorm.DB
	rdb        *redis.Client
	minioClient *minio.Client
	log        *logrus.Logger
	upgrader   = gorillaWs.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	// 并发控制
	msgChan    = make(chan interface{}, 1000)
	wg         sync.WaitGroup
	// RSA密钥
	rsaPublicKey  *rsa.PublicKey
	rsaPrivateKey *rsa.PrivateKey
)

var (
    // 注册登录IP限流存储 (key: ip地址)
    registerLoginIPLimiters = sync.Map{}
    // 注册登录用户限流存储 (key: 用户名/邮箱)
    registerLoginUserLimiters = sync.Map{}
    // 消息频率限流存储 (key: 客户端IP)
    messageConnLimiters = sync.Map{}
    // 房间用户限流存储 (key: fuid:group_id)
    GroupUserLimiters = sync.Map{}
)

// 数据库模型定义

// User 用户表
type User struct {
	ID        uint64 `gorm:"primarykey;autoIncrement"`
	FUID      string `gorm:"column:fuid;type:varchar(64);uniqueIndex;not null"`
	Username  string `gorm:"column:username;type:varchar(64);uniqueIndex;not null"`
	Nickname  string `gorm:"column:nickname;type:varchar(64);not null"`
	Email     string `gorm:"column:email;type:varchar(128);uniqueIndex;not null"`
	Password  string `gorm:"column:password;type:varchar(128);not null"` // bcrypt加密
	Avatar    string `gorm:"column:avatar;type:varchar(256);default:''"`
	Signature string `gorm:"column:signature;type:varchar(256);default:''"`
	VIPLevel  uint8  `gorm:"column:vip_level;type:tinyint;default:0"`
	VIPExp    uint64 `gorm:"column:vip_exp;type:bigint;default:0"`
	VIPStartTime time.Time `gorm:"column:vip_start_time;type:datetime;default:null"`
	Status    uint8  `gorm:"column:status;type:tinyint;default:1"` // 1:正常 0:禁用
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;autoUpdateTime"`
}

func (u *User) TableName() string {
	return "users"
}

// Friend 好友表
type Friend struct {
	ID        uint64 `gorm:"primarykey;autoIncrement"`
	UserFUID  string `gorm:"column:user_fuid;type:varchar(64);index;not null"` // 自己的fuid
	FriendFUID string `gorm:"column:friend_fuid;type:varchar(64);index;not null"` // 好友的fuid
	Remark    string `gorm:"column:remark;type:varchar(64);default:''"` // 备注
	Status    uint8  `gorm:"column:status;type:tinyint;default:1"` // 1:正常 2:黑名单 0:已删除
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;autoUpdateTime"`
}

func (f *Friend) TableName() string {
	return "friends"
}

// Group 群聊表
type Group struct {
	ID        uint64 `gorm:"primarykey;autoIncrement"`
	QUID      string `gorm:"column:quid;type:varchar(64);uniqueIndex;not null"`
	Name      string `gorm:"column:name;type:varchar(64);not null"`
	OwnerFUID string `gorm:"column:owner_fuid;type:varchar(64);index;not null"` // 群主fuid
	Avatar    string `gorm:"column:avatar;type:varchar(256);default:''"`
	Desc      string `gorm:"column:desc;type:varchar(256);default:''"`
	VIPLevel  uint8  `gorm:"column:vip_level;type:tinyint;default:0"`
	VIPExp    uint64 `gorm:"column:vip_exp;type:bigint;default:0"`
	VIPStartTime time.Time `gorm:"column:vip_start_time;type:datetime;default:null"`
	Status    uint8  `gorm:"column:status;type:tinyint;default:1"` // 1:正常 0:解散
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;autoUpdateTime"`
}

func (g *Group) TableName() string {
	return "groups"
}

// GroupMember 群成员表
type GroupMember struct {
	ID        uint64 `gorm:"primarykey;autoIncrement"`
	GroupQUID string `gorm:"column:group_quid;type:varchar(64);index;not null"` // 群quid
	UserFUID  string `gorm:"column:user_fuid;type:varchar(64);index;not null"` // 成员fuid
	Role      uint8  `gorm:"column:role;type:tinyint;default:0"` // 0:普通 1:群主 2:管理
	MuteEndTime time.Time `gorm:"column:mute_end_time;type:datetime;default:null"` // 禁言结束时间
	Status    uint8  `gorm:"column:status;type:tinyint;default:1"` // 1:正常 0:已退出/踢出
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;autoUpdateTime"`
}

func (gm *GroupMember) TableName() string {
	return "group_members"
}

// Message 消息表
type Message struct {
	ID        uint64 `gorm:"primarykey;autoIncrement"`
	MsgID     string `gorm:"column:msg_id;type:varchar(64);uniqueIndex;not null"` // 消息唯一ID
	SenderFUID string `gorm:"column:sender_fuid;type:varchar(64);index;not null"` // 发送者fuid
	ReceiverType uint8 `gorm:"column:receiver_type;type:tinyint;not null"` // 1:单聊 2:群聊
	ReceiverID string `gorm:"column:receiver_id;type:varchar(64);index;not null"` // 单聊:好友fuid 群聊:群quid
	ContentType uint8 `gorm:"column:content_type;type:tinyint;not null"` // 1:文字 2:图片 3:文件 4:表情 5:系统消息
	Content   string `gorm:"column:content;type:text;not null"` // 加密后的内容
	FontStyle string `gorm:"column:font_style;type:varchar(64);default:''"` // 字体样式
	FontSize  int    `gorm:"column:font_size;type:int;default:14"` // 字体大小
	FontColor string `gorm:"column:font_color;type:varchar(16);default:'#000000'"` // 字体颜色
	IsRecalled bool  `gorm:"column:is_recalled;type:tinyint;default:0"` // 是否撤回
	IsRead    bool  `gorm:"column:is_read;type:tinyint;default:0"` // 是否已读
	SendTime  time.Time `gorm:"column:send_time;type:datetime;not null"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;autoUpdateTime"`
}

func (m *Message) TableName() string {
	return "messages"
}

// SystemMessage 系统消息表
type SystemMessage struct {
	ID        uint64 `gorm:"primarykey;autoIncrement"`
	MsgID     string `gorm:"column:msg_id;type:varchar(64);uniqueIndex;not null"`
	Title     string `gorm:"column:title;type:varchar(64);not null"`
	Content   string `gorm:"column:content;type:text;not null"`
	TargetType uint8 `gorm:"column:target_type;type:tinyint;not null"` // 1:全体 2:指定用户 3:指定群
	TargetIDs string `gorm:"column:target_ids;type:text;default:''"` // 用户fuid/群quid列表，逗号分隔
	SendCount int    `gorm:"column:send_count;type:int;default:0"` // 已发送次数
	MaxSendCount int `gorm:"column:max_send_count;type:int;default:1"` // 最大发送次数
	CycleSend bool  `gorm:"column:cycle_send;type:tinyint;default:0"` // 是否循环发送
	FixedTime string `gorm:"column:fixed_time;type:varchar(8);default:''"` // 定点发送时间
	Status    uint8  `gorm:"column:status;type:tinyint;default:0"` // 0:待发送 1:发送中 2:已完成
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;autoUpdateTime"`
}

func (sm *SystemMessage) TableName() string {
	return "system_messages"
}

// OfflineMessage 离线消息表
type OfflineMessage struct {
	ID        uint64 `gorm:"primarykey;autoIncrement"`
	UserFUID  string `gorm:"column:user_fuid;type:varchar(64);index;not null"`
	MsgID     string `gorm:"column:msg_id;type:varchar(64);not null"`
	Status    uint8  `gorm:"column:status;type:tinyint;default:0"` // 0:未推送 1:已推送
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;autoCreateTime"`
}

func (om *OfflineMessage) TableName() string {
	return "offline_messages"
}

// GroupNotice 群公告表
type GroupNotice struct {
	ID        uint64 `gorm:"primarykey;autoIncrement"`
	GroupQUID string `gorm:"column:group_quid;type:varchar(64);index;not null"`
	Content   string `gorm:"column:content;type:text;not null"`
	PublisherFUID string `gorm:"column:publisher_fuid;type:varchar(64);not null"` // 发布者fuid
	PublishTime time.Time `gorm:"column:publish_time;type:datetime;not null"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;autoUpdateTime"`
}

func (gn *GroupNotice) TableName() string {
	return "group_notices"
}

// 通话记录模型
type Call struct {
	CallID        string    `gorm:"primaryKey;size:64" json:"call_id"` // 通话ID
	SenderFUID    string    `gorm:"size:64" json:"sender_fuid"`        // 发起者FUID
	ReceiverType  uint8     `json:"receiver_type"`                     // 1:单聊 2:群聊
	ReceiverID    string    `gorm:"size:64" json:"receiver_id"`        // 接收方ID
	CallType      uint8     `json:"call_type"`                         // 1:语音 2:视频
	Status        uint8     `json:"status"`                            // 0:等待接听 1:通话中 2:已拒绝 3:已结束 4:未接听
	StartTime     time.Time `json:"start_time"`                        // 开始时间
	EndTime       time.Time `json:"end_time"`                          // 结束时间
	Duration      int       `json:"duration"`                          // 通话时长(秒)
	CreateTime    time.Time `gorm:"autoCreateTime" json:"create_time"`
}

// Device 设备信息表
type Device struct {
	ID           uint64    `gorm:"primarykey;autoIncrement"`
	UserFUID     string    `gorm:"column:user_fuid;type:varchar(64);index;not null"` // 所属用户FUID
	DeviceID     string    `gorm:"column:device_id;type:varchar(64);index;not null"` // 设备唯一标识
	DeviceName   string    `gorm:"column:device_name;type:varchar(64);not null"`     // 设备名称(如"iPhone 13"、"Chrome浏览器")
	DeviceType   string    `gorm:"column:device_type;type:varchar(32);not null"`     // 设备类型(phone/pc/web/app)
	LoginIP      string    `gorm:"column:login_ip;type:varchar(64);not null"`        // 登录IP
	LoginTime    time.Time `gorm:"column:login_time;type:datetime;not null"`         // 登录时间
	LastActive   time.Time `gorm:"column:last_active;type:datetime;not null"`        // 最后活跃时间
	RefreshToken string    `gorm:"column:refresh_token;type:varchar(256);not null"`  // 刷新令牌
	Status       uint8     `gorm:"column:status;type:tinyint;default:1"`             // 1:在线 0:离线
	CreatedAt    time.Time `gorm:"column:created_at;type:datetime;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;type:datetime;autoUpdateTime"`
}

func (d *Device) TableName() string {
	return "devices"
}

// 扩展JWT Claims以包含设备信息
type CustomClaims struct {
	FUID     string `json:"fuid"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	DeviceID string `json:"device_id"` // 新增设备ID
	jwt.RegisteredClaims
}

// 加载配置文件
func loadConfig() error {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return fmt.Errorf("read config file failed: %v", err)
	}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return fmt.Errorf("unmarshal config file failed: %v", err)
	}
	return nil
}

// 初始化日志
func initLog() error {
	log = logrus.New()
	// 创建日志目录
	if err := os.MkdirAll(cfg.Log.Path, 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("create log dir failed: %v", err)
	}
	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Log.Level)
	if err != nil {
		log.SetLevel(logrus.InfoLevel)
	} else {
		log.SetLevel(level)
	}
	// 设置日志格式
	log.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})
	// 日志文件分割配置
	logFileName := filepath.Join(cfg.Log.Path, cfg.Log.FileNameFormat)
	// 输出到控制台和文件
	log.SetOutput(&lumberjack.Logger{
    Filename:   logFileName,
    MaxSize:    cfg.Log.MaxSize,    // MB
    MaxBackups: cfg.Log.MaxBackups,
    MaxAge:     cfg.Log.MaxAge,     // 天
    Compress:   cfg.Log.Compress,
	})
	return nil
	}

// 初始化MySQL连接
func initMySQL() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		cfg.MySQL.User,
		cfg.MySQL.Passwd,
		cfg.MySQL.Host,
		cfg.MySQL.Port,
		cfg.MySQL.Database,
		cfg.MySQL.Charset,
	)
	// 设置日志级别
	var logLevel logger.LogLevel
	if cfg.App.Mode == "debug" {
		logLevel = logger.Info
	} else {
		logLevel = logger.Error
	}
	// 初始化gorm
	gormDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // 单数表名
		},
	})
	if err != nil {
		return fmt.Errorf("connect mysql failed: %v", err)
	}
	// 设置连接池
	sqlDB, err := gormDB.DB()
	if err != nil {
		return fmt.Errorf("get sql db failed: %v", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.MySQL.ConnMaxLifetime) * time.Second)
	// 自动迁移表
	// err = gormDB.AutoMigrate(
	//	&User{},
	//	&Friend{},
	//	&Group{},
	//	&GroupMember{},
	//	&Message{},
	//	&SystemMessage{},
	//	&OfflineMessage{},
	//	&GroupNotice{},
	// )
	if err != nil {
		return fmt.Errorf("auto migrate tables failed: %v", err)
	}
	db = gormDB
	log.Info("MySQL服务连接成功")
	return nil
}

// 初始化Redis连接
func initRedis() error {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		IdleTimeout:  time.Duration(cfg.Redis.IdleTimeout) * time.Second,
	})
	// 测试连接
    ctx := context.Background()
    _, err := client.Ping(ctx).Result()
    if err != nil {
        return fmt.Errorf("connect redis failed: %v", err)
    }
    rdb = client
	log.Info("Redis服务连接成功")
	return nil
}

// 初始化MinIO客户端
func initMinIO() error {
	if cfg.Storage.Type != "minio" {
		return nil
	}
	client, err := minio.New(cfg.Storage.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Storage.MinIO.AccessKey, cfg.Storage.MinIO.SecretKey, ""),
		Secure: cfg.Storage.MinIO.UseSSL,
	})
	if err != nil {
		return fmt.Errorf("connect minio failed: %v", err)
	}
	// 检查桶是否存在
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Storage.MinIO.Bucket)
	if err != nil {
		return fmt.Errorf("check bucket exists failed: %v", err)
	}
	if !exists {
		// 创建桶
		err = client.MakeBucket(ctx, cfg.Storage.MinIO.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("create bucket failed: %v", err)
		}
	}
	minioClient = client
	log.Info("MinIO初始化成功")
	return nil
}

// 初始化RSA密钥
func initRSA() error {
    // 读取公钥文件
    publicKeyData, err := os.ReadFile(cfg.Crypto.RSA.PublicKeyPath)
    if err != nil {
        return fmt.Errorf("读取公钥文件失败: %v", err)
    }
    // 解析公钥
    publicBlock, _ := pem.Decode(publicKeyData)
    if publicBlock == nil {
        return errors.New("解析公钥PEM格式失败")
    }
    pubKey, err := x509.ParsePKIXPublicKey(publicBlock.Bytes)
    if err != nil {
        return fmt.Errorf("解析公钥内容失败: %v", err)
    }
    rsaPublicKey = pubKey.(*rsa.PublicKey)

    // 读取私钥文件
    privateKeyData, err := os.ReadFile(cfg.Crypto.RSA.PrivateKeyPath)
    if err != nil {
        return fmt.Errorf("读取私钥文件失败: %v", err)
    }
    // 解析私钥
    privateBlock, _ := pem.Decode(privateKeyData)
    if privateBlock == nil {
        return errors.New("解析私钥PEM格式失败")
    }
    privKey, err := x509.ParsePKCS8PrivateKey(privateBlock.Bytes)
    if err != nil {
        return fmt.Errorf("解析私钥内容失败: %v", err)
    }
    rsaPrivateKey = privKey.(*rsa.PrivateKey)

    log.Info("RSA秘钥加载成功")
    return nil
}

// 校验并加载TLS证书（仅生产模式）
func loadTLSCerts() (tls.Certificate, error) {
	// 检查证书文件是否存在
	if _, err := os.Stat(cfg.App.TLS.CertPath); os.IsNotExist(err) {
		return tls.Certificate{}, fmt.Errorf("证书文件不存在: %s", cfg.App.TLS.CertPath)
	}
	if _, err := os.Stat(cfg.App.TLS.KeyPath); os.IsNotExist(err) {
		return tls.Certificate{}, fmt.Errorf("私钥文件不存在: %s", cfg.App.TLS.KeyPath)
	}

	// 加载证书
	cert, err := tls.LoadX509KeyPair(cfg.App.TLS.CertPath, cfg.App.TLS.KeyPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("证书加载失败: %v（可能是证书格式错误或密钥不匹配）", err)
	}

	// 简单校验证书有效性（检查有效期）
	now := time.Now()
	for _, cert := range cert.Certificate {
		x509Cert, err := x509.ParseCertificate(cert)
		if err != nil {
			return tls.Certificate{}, fmt.Errorf("证书解析失败: %v", err)
		}
		if now.Before(x509Cert.NotBefore) {
			return tls.Certificate{}, fmt.Errorf("证书未生效，生效时间: %s", x509Cert.NotBefore.Format(time.RFC3339))
		}
		if now.After(x509Cert.NotAfter) {
			return tls.Certificate{}, fmt.Errorf("证书已过期，过期时间: %s", x509Cert.NotAfter.Format(time.RFC3339))
		}
	}

	return cert, nil
}

// 生成唯一FUID
func generateFUID() (string, error) {
	return generateUniqueID(cfg.Business.User.FUIDLen)
}

// 生成唯一QUID
func generateQUID() (string, error) {
	return generateUniqueID(cfg.Business.Group.QUIDLen)
}

// 生成唯一ID
func generateUniqueID(length int) (string, error) {
	const charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	id := string(b)
	// 检查唯一性（FUID/QUID）
	// 先查Redis缓存
	ctx := context.Background()
	set, err := rdb.SetNX(ctx,"unique_id:"+id, 1, time.Hour).Result()
	if err != nil {
		return "", err
	}
	if !set {
		return generateUniqueID(length)
	}
	return id, nil
}

// 密码加密（bcrypt）
func encryptPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cfg.Business.User.PasswordCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// 验证密码
func verifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// AES256加密（JWT签名）
func aesEncrypt(plainText, key []byte) ([]byte, error) {
		// 校验密钥长度
	switch len(key) {
	case 16, 24, 32:
	default:
		return nil, fmt.Errorf("AES密钥长度必须为16、24或32字节，当前为%d字节", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	plainText = pkcs7Padding(plainText, blockSize)
	cipherText := make([]byte, len(plainText))
	iv := make([]byte, blockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherText, plainText)
	return append(iv, cipherText...), nil
}

// AES256解密
func aesDecrypt(cipherText, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	if len(cipherText) < blockSize {
		return nil, errors.New("cipher text too short")
	}
	iv := cipherText[:blockSize]
	cipherText = cipherText[blockSize:]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(cipherText, cipherText)
	return pkcs7Unpadding(cipherText)
}

// PKCS7填充
func pkcs7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// PKCS7解填充
func pkcs7Unpadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("padding data is empty")
	}
	padding := int(data[length-1])
	if padding > length || padding == 0 {
		return nil, errors.New("invalid padding")
	}
	return data[:length-padding], nil
}

// RSA加密
func rsaEncrypt(data []byte) ([]byte, error) {
	cipherText, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPublicKey, data, nil)
	if err != nil {
		return nil, err
	}
	return []byte(base64.StdEncoding.EncodeToString(cipherText)), nil
}

// RSA解密
func rsaDecrypt(data []byte) ([]byte, error) {
	decodedData, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}
	plainText, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, rsaPrivateKey, decodedData, nil)
	if err != nil {
		return nil, err
	}
	return plainText, nil
}

// 获取客户端真实IP
func getRealIP(c *gin.Context) string {
	// 常见CDN代理头
	headers := []string{
		"X-Forwarded-For",
		"X-Real-IP",
		"CF-Connecting-IP",
		"X-Forwarded",
		"Forwarded-For",
		"Forwarded",
		"Proxy-Client-IP",
		"WL-Proxy-Client-IP",
	}
	for _, header := range headers {
		ipStr := c.GetHeader(header)
		if ipStr != "" {
			// 多个IP时取第一个
			ips := strings.Split(ipStr, ",")
			ip := strings.TrimSpace(ips[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}
	// 最后取远程IP
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return ip
}

// 健康检查接口
func healthHandler(c *gin.Context) {
    // 计算服务已运行时间
    uptime := time.Since(startTime)
    
    // 格式化运行时间为友好显示格式
    uptimeStr := fmt.Sprintf(
        "%d小时%d分钟%d秒",
        int(uptime.Hours()),
        int(uptime.Minutes())%60,
        int(uptime.Seconds())%60,
    )
    
    // 构建响应数据
    data := map[string]interface{}{
        "status":  "success",
        "message": "服务运行正常",
        "uptime":  uptimeStr,
        "start_at": startTime.Format("2006-01-02 15:04:05"),
    }
    
    success(c, data)
}

// 创建基于令牌桶的限流器
// rate: 每秒生成的令牌数
// burst: 令牌桶容量
func newTokenBucketLimiter(r rate.Limit, burst int) *rate.Limiter {
    return rate.NewLimiter(r, burst)
}

// 获取注册登录IP限流器
func getRegisterLoginIPLimiter(ip string) *rate.Limiter {
    if limiter, ok := registerLoginIPLimiters.Load(ip); ok {
        return limiter.(*rate.Limiter)
    }
    
    // 从配置转换：周期内限制数 -> 每秒令牌数
    limit := cfg.RateLimit.RegisterLoginIP.Limit
    period := cfg.RateLimit.RegisterLoginIP.Period
    r := rate.Limit(limit) / rate.Limit(period) // 每秒生成的令牌数
    burst := limit                             // 令牌桶容量
    
    newLimiter := newTokenBucketLimiter(r, burst)
    limiter, _ := registerLoginIPLimiters.LoadOrStore(ip, newLimiter)
    return limiter.(*rate.Limiter)
}

// 获取注册登录用户限流器
func getRegisterLoginUserLimiter(username string) *rate.Limiter {
    if limiter, ok := registerLoginUserLimiters.Load(username); ok {
        return limiter.(*rate.Limiter)
    }
    
    limit := cfg.RateLimit.RegisterLoginUser.Limit
    period := cfg.RateLimit.RegisterLoginUser.Period
    r := rate.Limit(limit) / rate.Limit(period)
    burst := limit
    
    newLimiter := newTokenBucketLimiter(r, burst)
    limiter, _ := registerLoginUserLimiters.LoadOrStore(username, newLimiter)
    return limiter.(*rate.Limiter)
}

// 获取消息频率限流器
func getMessageConnLimiter(ip string) *rate.Limiter {
    if limiter, ok := messageConnLimiters.Load(ip); ok {
        return limiter.(*rate.Limiter)
    }
    
    limit := cfg.RateLimit.MessageConn.Limit
    period := cfg.RateLimit.MessageConn.Period
    r := rate.Limit(limit) / rate.Limit(period)
    burst := limit
    
    newLimiter := newTokenBucketLimiter(r, burst)
    limiter, _ := messageConnLimiters.LoadOrStore(ip, newLimiter)
    return limiter.(*rate.Limiter)
}

// 获取房间用户限流器
func getGroupUserLimiter(key string) *rate.Limiter {
    if limiter, ok := GroupUserLimiters.Load(key); ok {
        return limiter.(*rate.Limiter)
    }
    
    limit := cfg.RateLimit.GroupUser.Limit
    period := cfg.RateLimit.GroupUser.Period
    r := rate.Limit(limit) / rate.Limit(period)
    burst := limit
    
    newLimiter := newTokenBucketLimiter(r, burst)
    limiter, _ := GroupUserLimiters.LoadOrStore(key, newLimiter)
    return limiter.(*rate.Limiter)
}

// 初始化限流中间件（基于官方令牌桶算法）
func initRateLimiters() map[string]gin.HandlerFunc {
    limiters := make(map[string]gin.HandlerFunc)
    
    // 注册/登录IP限流中间件
    limiters["register_login_ip"] = func(c *gin.Context) {
        ip := getRealIP(c)
        limiter := getRegisterLoginIPLimiter(ip)
        if !limiter.Allow() {
            fail(c, 429, "IP请求过于频繁，请稍后再试")
            c.Abort()
            return
        }
        c.Next()
    }
    
    // 注册/登录用户限流中间件
    limiters["register_login_user"] = func(c *gin.Context) {
        username := c.PostForm("username")
        if username == "" {
            username = c.PostForm("email")
        }
        if username == "" {
            fail(c, 400, "用户名或邮箱不能为空")
            c.Abort()
            return
        }
        
        limiter := getRegisterLoginUserLimiter(username)
        if !limiter.Allow() {
            fail(c, 429, "账号请求过于频繁，请稍后再试")
            c.Abort()
            return
        }
        c.Next()
    }
    
    // 消息频率限流中间件
    limiters["message_conn"] = func(c *gin.Context) {
        ip := c.ClientIP()
        limiter := getMessageConnLimiter(ip)
        if !limiter.Allow() {
            fail(c, 429, "消息发送过于频繁，请稍后再试")
            c.Abort()
            return
        }
        c.Next()
    }
    
    // 房间/用户维度限流中间件
    limiters["group_user"] = func(c *gin.Context) {
        fuid := c.GetHeader("FUID")
        groupID := c.PostForm("group_id")
        if fuid == "" || groupID == "" {
            fail(c, 400, "FUID或房间ID不能为空")
            c.Abort()
            return
        }
        key := fmt.Sprintf("%s:%s", fuid, groupID)
        
        limiter := getGroupUserLimiter(key)
        if !limiter.Allow() {
            fail(c, 429, "房间内操作过于频繁，请稍后再试")
            c.Abort()
            return
        }
        c.Next()
    }
    
    return limiters
}

// 标准化响应结构体
type Response struct {
	Code    int         `json:"code"`
	Msg     string      `json:"msg"`
	Data    interface{} `json:"data,omitempty"`
	Count   int64       `json:"count,omitempty"`
}

// 成功响应
func success(c *gin.Context, data interface{}, count ...int64) {
	res := Response{
		Code: 200,
		Msg:  "success",
		Data: data,
	}
	if len(count) > 0 {
		res.Count = count[0]
	}
	c.JSON(200, res)
}

// 错误响应
func fail(c *gin.Context, code int, msg string) {
	res := Response{
		Code: code,
		Msg:  msg,
	}
	c.JSON(200, res)
	log.Errorf("API error: code=%d, msg=%s, path=%s, ip=%s", code, msg, c.Request.URL.Path, getRealIP(c))
}

// 验证CF人机验证
func verifyCFTurnstile(token string) bool {
	if cfg.App.Mode == "debug" {
		return true // 调试模式跳过验证
	}
	resp, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify",
		url.Values{
			"secret":   {cfg.CFTurnstile.SecretKey},
			"response": {token},
		})
	if err != nil {
		log.Error("verify CF turnstile failed: ", err)
		return false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("read CF turnstile response failed: ", err)
		return false
	}
	var result struct {
		Success bool `json:"success"`
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Error("unmarshal CF turnstile response failed: ", err)
		return false
	}
	return result.Success
}

// 用户注册接口
func registerHandler(c *gin.Context) {
	// 参数绑定
	var req struct {
		Username        string `json:"username" binding:"required,min=3,max=20"`
		Nickname        string `json:"nickname" binding:"required,min=1,max=20"`
		Email           string `json:"email" binding:"required,email"`
		Password        string `json:"password" binding:"required,min=6,max=20"`
		ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=Password"`
		TurnstileToken  string `json:"turnstile_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: "+err.Error())
		return
	}
	// 验证人机验证
	if !verifyCFTurnstile(req.TurnstileToken) {
		fail(c, 403, "人机验证失败")
		return
	}
	// 检查用户名是否已存在
	var user User
	if err := db.Where("username = ?", req.Username).First(&user).Error; err == nil {
		fail(c, 400, "用户名已存在")
		return
	}
	// 检查邮箱是否已存在
	if err := db.Where("email = ?", req.Email).First(&user).Error; err == nil {
		fail(c, 400, "邮箱已存在")
		return
	}
	// 生成FUID
	fuid, err := generateFUID()
	if err != nil {
		fail(c, 500, "生成用户ID失败，请稍后重试")
		log.Errorf("生成FUID失败: %v", err)
		return
	}
	// 加密密码
	hashPassword, err := encryptPassword(req.Password)
	if err != nil {
		fail(c, 500, "密码加密失败: "+err.Error())
		return
	}
	// 创建用户
	newUser := User{
		FUID:     fuid,
		Username: req.Username,
		Nickname: req.Nickname,
		Email:    req.Email,
		Password: hashPassword,
	}
	if err := db.Create(&newUser).Error; err != nil {
		fail(c, 500, "创建用户失败: "+err.Error())
		return
	}
	// 返回用户信息（隐藏敏感信息）
	respData := map[string]interface{}{
		"fuid":    newUser.FUID,
		"username": newUser.Username,
		"nickname": newUser.Nickname,
		"email":    newUser.Email,
	}
	success(c, respData)
	log.Infof("User registered: fuid=%s, username=%s, email=%s", newUser.FUID, newUser.Username, newUser.Email)
}

// 用户登录接口 - 支持多设备
func loginHandler(c *gin.Context) {
	// 参数绑定，新增设备信息
	var req struct {
		Account         string `json:"account" binding:"required"` // 用户名/邮箱/FUID
		Password        string `json:"password" binding:"required"`
		TurnstileToken  string `json:"turnstile_token" binding:"required"`
		DeviceName      string `json:"device_name" binding:"required"` // 设备名称
		DeviceType      string `json:"device_type" binding:"required,oneof=phone pc web app"` // 设备类型
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: "+err.Error())
		return
	}

	// 验证人机验证
	if !verifyCFTurnstile(req.TurnstileToken) {
		fail(c, 403, "人机验证失败")
		return
	}

	// 查询用户
	var user User
	condition := "username = ? OR email = ? OR fuid = ?"
	if err := db.Where(condition, req.Account, req.Account, req.Account).First(&user).Error; err != nil {
		fail(c, 401, "账号或密码错误")
		return
	}

	// 检查用户状态
	if user.Status == 0 {
		fail(c, 403, "账号已被禁用")
		return
	}

	// 验证密码
	if !verifyPassword(req.Password, user.Password) {
		fail(c, 401, "账号或密码错误")
		return
	}

	// 生成设备ID
	deviceID := generateDeviceID()
	loginIP := getRealIP(c)
	now := time.Now()

	// 生成刷新令牌
	refreshToken, err := generateRefreshToken(user.FUID, deviceID)
	if err != nil {
		fail(c, 500, "生成令牌失败")
		return
	}

	// 保存设备信息
	device := Device{
		UserFUID:     user.FUID,
		DeviceID:     deviceID,
		DeviceName:   req.DeviceName,
		DeviceType:   req.DeviceType,
		LoginIP:      loginIP,
		LoginTime:    now,
		LastActive:   now,
		RefreshToken: refreshToken,
		Status:       1, // 在线状态
	}
	if err := db.Create(&device).Error; err != nil {
		fail(c, 500, "保存设备信息失败")
		return
	}

	// 生成访问令牌
	accessExp := now.Add(time.Duration(cfg.Crypto.JWT.Expire) * time.Second)
	claims := CustomClaims{
		FUID:     user.FUID,
		Username: user.Username,
		Nickname: user.Nickname,
		DeviceID: deviceID, // 包含设备ID
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.Crypto.JWT.Secret))
	if err != nil {
		fail(c, 500, "生成访问令牌失败")
		return
	}

	// 存储设备在线状态到Redis
	ctx := context.Background()
	deviceKey := fmt.Sprintf("user:devices:%s", user.FUID)
	deviceInfo, _ := json.Marshal(map[string]interface{}{
		"device_id":   deviceID,
		"device_name": req.DeviceName,
		"device_type": req.DeviceType,
		"login_ip":    loginIP,
		"login_time":  now.Format(time.RFC3339),
		"status":      1,
	})
	rdb.HSet(ctx, deviceKey, deviceID, deviceInfo)
	// 设置过期时间(与refresh token一致)
	rdb.Expire(ctx, deviceKey, time.Duration(cfg.Crypto.JWT.RefreshExpire)*time.Second)

	// 返回登录信息
	success(c, map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    cfg.Crypto.JWT.Expire,
		"device_id":     deviceID,
		"user": map[string]interface{}{
			"fuid":     user.FUID,
			"username": user.Username,
			"nickname": user.Nickname,
		},
	})

	log.Infof("User logged in: fuid=%s, device_id=%s, ip=%s", user.FUID, deviceID, loginIP)
}

// 验证JWT中间件
func authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 获取Token
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            fail(c, 401, "未提供令牌")
            c.Abort()
            return
        }

        // 检查Bearer前缀
        parts := strings.SplitN(authHeader, " ", 2)
        if !(len(parts) == 2 && parts[0] == "Bearer") {
            fail(c, 401, "令牌格式错误（需Bearer前缀）")
            c.Abort()
            return
        }
        tokenStr := parts[1]

        // 解析Token
        token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
            // 验证签名方法
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("不支持的签名方法: %v", token.Header["alg"])
            }
            // 返回签名密钥
            return []byte(cfg.Crypto.JWT.Secret), nil
        })

        // 处理解析错误
        if err != nil {
            var msg string
            switch {
            case errors.Is(err, jwt.ErrSignatureInvalid):
                msg = "令牌签名无效"
            case errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet):
                msg = "令牌已过期或未生效"
            default:
                msg = "令牌解析失败"
            }
            fail(c, 401, msg)
            log.Errorf("JWT验证失败: %v, token: %s", err, tokenStr)
            c.Abort()
            return
        }

        // 验证claims有效性
        if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
            // 额外校验签发者（确保令牌来源正确）
            if claims.Issuer != cfg.App.Name {
                fail(c, 401, "无效的令牌签发者")
                c.Abort()
                return
            }

            // 校验令牌主题（确保是访问令牌）
            if claims.Subject != "access_token" {
                fail(c, 401, "令牌类型错误")
                c.Abort()
                return
            }

            // 检查用户是否存在（增强安全性）
            var user User
            if err := db.Where("fuid = ? AND status = 1", claims.FUID).First(&user).Error; err != 			nil {
                fail(c, 401, "用户不存在或已被禁用")
                c.Abort()
                return
            }

		if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
            // 将用户信息存入上下文
            c.Set("fuid", claims.FUID)
            c.Set("username", claims.Username)
            c.Set("nickname", claims.Nickname)
            c.Next()
        } else {
            fail(c, 401, "令牌无效")
            c.Abort()
            return
			}
		}
	}
}

// 刷新访问令牌接口
func refreshTokenHandler(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: "+err.Error())
		return
	}

	// 解析刷新令牌
	token, err := jwt.Parse(req.RefreshToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("不支持的签名方法")
		}
		return []byte(cfg.Crypto.JWT.Secret), nil
	})
	if err != nil || !token.Valid {
		fail(c, 401, "刷新令牌无效")
		return
	}

	// 提取令牌信息
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		fail(c, 401, "令牌格式错误")
		return
	}

	fuid, _ := claims["fuid"].(string)
	deviceID, _ := claims["device_id"].(string)

	// 验证设备状态
	var device Device
	if err := db.Where("user_fuid = ? AND device_id = ? AND refresh_token = ? AND status = 1",
		fuid, deviceID, req.RefreshToken).First(&device).Error; err != nil {
		fail(c, 401, "设备已下线或令牌无效")
		return
	}

	// 更新最后活跃时间
	now := time.Now()
	db.Model(&device).Update("last_active", now)

	// 生成新的访问令牌
	accessExp := now.Add(time.Duration(cfg.Crypto.JWT.Expire) * time.Second)
	newClaims := CustomClaims{
		FUID:     fuid,
		Username: device.UserFUID, // 实际应从用户表查询
		DeviceID: deviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims).SignedString([]byte(cfg.Crypto.JWT.Secret))
	if err != nil {
		fail(c, 500, "生成访问令牌失败")
		return
	}

	// 更新Redis中的最后活跃时间
	ctx := context.Background()
	deviceKey := fmt.Sprintf("user:devices:%s", fuid)
	existingInfo, _ := rdb.HGet(ctx, deviceKey, deviceID).Result()
	var deviceInfo map[string]interface{}
	json.Unmarshal([]byte(existingInfo), &deviceInfo)
	deviceInfo["last_active"] = now.Format(time.RFC3339)
	updatedInfo, _ := json.Marshal(deviceInfo)
	rdb.HSet(ctx, deviceKey, deviceID, updatedInfo)

	success(c, map[string]interface{}{
		"access_token": accessToken,
		"expires_in":   cfg.Crypto.JWT.Expire,
	})
}

// 生成设备唯一标识
func generateDeviceID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// 生成刷新令牌
func generateRefreshToken(fuid, deviceID string) (string, error) {
	// 刷新令牌包含用户ID和设备ID，有效期更长
	claims := jwt.MapClaims{
		"fuid":     fuid,
		"device_id": deviceID,
		"exp":      time.Now().Add(time.Duration(cfg.Crypto.JWT.RefreshExpire) * time.Second).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Crypto.JWT.Secret))
}

// 获取当前登录设备列表
func getDevicesHandler(c *gin.Context) {
	fuid := c.GetHeader("FUID") // 从JWT中间件获取
	if fuid == "" {
		fail(c, 401, "未授权")
		return
	}

	// 从Redis获取设备列表
	ctx := context.Background()
	deviceKey := fmt.Sprintf("user:devices:%s", fuid)
	devicesMap, err := rdb.HGetAll(ctx, deviceKey).Result()
	if err != nil {
		fail(c, 500, "获取设备列表失败")
		return
	}

	var devices []interface{}
	for _, infoStr := range devicesMap {
		var deviceInfo map[string]interface{}
		json.Unmarshal([]byte(infoStr), &deviceInfo)
		devices = append(devices, deviceInfo)
	}

	success(c, devices)
}

// 踢下线指定设备
func kickDeviceHandler(c *gin.Context) {
	fuid := c.GetHeader("FUID")
	if fuid == "" {
		fail(c, 401, "未授权")
		return
	}

	var req struct {
		DeviceID string `json:"device_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: "+err.Error())
		return
	}

	// 更新设备状态为离线
	ctx := context.Background()
	result := db.Model(&Device{}).
		Where("user_fuid = ? AND device_id = ?", fuid, req.DeviceID).
		Updates(map[string]interface{}{
			"status":       0,
			"refresh_token": "", // 失效刷新令牌
		})
	if result.Error != nil {
		fail(c, 500, "操作失败")
		return
	}
	if result.RowsAffected == 0 {
		fail(c, 404, "设备不存在")
		return
	}

	// 从Redis移除设备
	deviceKey := fmt.Sprintf("user:devices:%s", fuid)
	rdb.HDel(ctx, deviceKey, req.DeviceID)

	success(c, nil)
	log.Infof("Device kicked: fuid=%s, device_id=%s", fuid, req.DeviceID)
}

// JWT验证中间件
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			fail(c, 401, "请提供授权令牌")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			fail(c, 401, "授权令牌格式错误")
			c.Abort()
			return
		}

		// 解析token
		token, err := jwt.ParseWithClaims(parts[1], &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.Crypto.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			fail(c, 401, "无效的授权令牌")
			c.Abort()
			return
		}

		// 验证设备状态
		claims, ok := token.Claims.(*CustomClaims)
		if !ok {
			fail(c, 401, "令牌信息错误")
			c.Abort()
			return
		}

		// 检查设备是否在线
		var device Device
		if err := db.Where("user_fuid = ? AND device_id = ? AND status = 1",
			claims.FUID, claims.DeviceID).First(&device).Error; err != nil {
			fail(c, 401, "设备已下线，请重新登录")
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("FUID", claims.FUID)
		c.Set("Username", claims.Username)
		c.Set("DeviceID", claims.DeviceID)
		c.Next()
	}
}

// 搜索好友接口
func searchFriendHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 获取搜索参数
	keyword := c.Query("keyword")
	if keyword == "" {
		fail(c, 400, "搜索关键词不能为空")
		return
	}
	// 搜索用户（FUID/用户名/昵称/邮箱）
	var users []User
	condition := "(fuid LIKE ? OR username LIKE ? OR nickname LIKE ? OR email LIKE ?) AND status = 1"
	likeKeyword := "%" + keyword + "%"
	if err := db.Where(condition, likeKeyword, likeKeyword, likeKeyword, likeKeyword).
		Select("fuid, username, nickname, avatar, signature, vip_level, vip_exp").
		Find(&users).Error; err != nil {
		fail(c, 500, "搜索好友失败: "+err.Error())
		return
	}
	// 排除自己
	var result []map[string]interface{}
	for _, user := range users {
		if user.FUID == currentFUID {
			continue
		}
		// 检查是否已添加好友
		var friend Friend
		err := db.Where("user_fuid = ? AND friend_fuid = ? AND status != 0", currentFUID, user.FUID).First(&friend).Error
		isFriend := err == nil
		isBlack := false
		if isFriend {
			isBlack = friend.Status == 2
		}
		result = append(result, map[string]interface{}{
			"fuid":        user.FUID,
			"username":    user.Username,
			"nickname":    user.Nickname,
			"avatar":      user.Avatar,
			"signature":   user.Signature,
			"vip_level":   user.VIPLevel,
			"vip_exp":     user.VIPExp,
			"is_friend":   isFriend,
			"is_black":    isBlack,
		})
	}
	success(c, result, int64(len(result)))
}

// 添加好友接口
func addFriendHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 参数绑定
	var req struct {
		FriendFUID string `json:"friend_fuid" binding:"required"`
		Remark     string `json:"remark" binding:"max=20"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: "+err.Error())
		return
	}
	// 检查好友是否存在
	var friendUser User
	if err := db.Where("fuid = ? AND status = 1", req.FriendFUID).First(&friendUser).Error; err != nil {
		fail(c, 400, "好友不存在或已被禁用")
		return
	}
	// 检查是否是自己
	if req.FriendFUID == currentFUID {
		fail(c, 400, "不能添加自己为好友")
		return
	}
	// 检查是否已添加
	var friend Friend
	err := db.Where("user_fuid = ? AND friend_fuid = ?", currentFUID, req.FriendFUID).First(&friend).Error
	if err == nil {
		if friend.Status == 1 {
			fail(c, 400, "已添加该用户为好友")
		} else if friend.Status == 2 {
			fail(c, 400, "该用户在你的黑名单中，请先移除")
		} else if friend.Status == 0 {
			// 恢复好友关系
			friend.Status = 1
			friend.Remark = req.Remark
			if err := db.Save(&friend).Error; err != nil {
				fail(c, 500, "恢复好友关系失败: "+err.Error())
				return
			}
			success(c, map[string]string{"msg": "恢复好友关系成功"})
			return
		}
	}
	// 检查好友数量是否超限
	var friendCount int64
	db.Model(&Friend{}).Where("user_fuid = ? AND status = 1", currentFUID).Count(&friendCount)
	if friendCount >= int64(cfg.Business.User.FriendMax) {
		fail(c, 400, fmt.Sprintf("好友数量已达上限(%d)", cfg.Business.User.FriendMax))
		return
	}
	// 添加好友
	newFriend := Friend{
		UserFUID:    currentFUID,
		FriendFUID:  req.FriendFUID,
		Remark:      req.Remark,
		Status:      1,
	}
	if err := db.Create(&newFriend).Error; err != nil {
		fail(c, 500, "添加好友失败: "+err.Error())
		return
	}
	
	// 同时需要为对方添加反向好友关系
	reverseFriend := Friend{
    UserFUID:    req.FriendFUID,
    FriendFUID:  currentFUID,
    Remark:      "",
    Status:      1,
	}
	
	if err := db.Create(&reverseFriend).Error; err != nil {
    // 回滚当前好友记录
    db.Delete(&newFriend)
    fail(c, 500, "添加好友失败: "+err.Error())
    return
	}

	// 发送ntfy推送（如果启用）
	if cfg.Business.Notify.Ntfy.Enable {
		go sendNtfyNotification("添加好友通知", fmt.Sprintf("用户%s(%s)添加你为好友", 
			c.GetString("nickname"), currentFUID), req.FriendFUID)
	}
	success(c, map[string]string{"msg": "添加好友成功"})
	log.Infof("Add friend: user=%s, friend=%s", currentFUID, req.FriendFUID)
}

// 删除好友接口
func deleteFriendHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 获取好友FUID
	friendFUID := c.Param("fuid")
	if friendFUID == "" {
		fail(c, 400, "好友FUID不能为空")
		return
	}
	// 更新好友状态为已删除
	err := db.Model(&Friend{}).Where("user_fuid = ? AND friend_fuid = ?", currentFUID, friendFUID).
		Update("status", 0).Error
	if err != nil {
		fail(c, 500, "删除好友失败: "+err.Error())
		return
	}
	success(c, map[string]string{"msg": "删除好友成功"})
	log.Infof("Delete friend: user=%s, friend=%s", currentFUID, friendFUID)
}

// 修改好友备注接口
func updateFriendRemarkHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 参数绑定
	var req struct {
		FriendFUID string `json:"friend_fuid" binding:"required"`
		Remark     string `json:"remark" binding:"max=20"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: "+err.Error())
		return
	}
	// 更新备注
	err := db.Model(&Friend{}).Where("user_fuid = ? AND friend_fuid = ? AND status = 1", 
		currentFUID, req.FriendFUID).Update("remark", req.Remark).Error
	if err != nil {
		fail(c, 500, "修改备注失败: "+err.Error())
		return
	}
	success(c, map[string]string{"msg": "修改备注成功"})
	log.Infof("Update friend remark: user=%s, friend=%s, remark=%s", currentFUID, req.FriendFUID, req.Remark)
}

// 加入黑名单接口
func addBlacklistHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 获取好友FUID
	friendFUID := c.Param("fuid")
	if friendFUID == "" {
		fail(c, 400, "好友FUID不能为空")
		return
	}
	// 更新好友状态为黑名单
	err := db.Model(&Friend{}).Where("user_fuid = ? AND friend_fuid = ? AND status = 1", 
		currentFUID, friendFUID).Update("status", 2).Error
	if err != nil {
		fail(c, 500, "加入黑名单失败: "+err.Error())
		return
	}
	success(c, map[string]string{"msg": "加入黑名单成功"})
	log.Infof("Add blacklist: user=%s, friend=%s", currentFUID, friendFUID)
}

// 移出黑名单接口
func removeBlacklistHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 获取好友FUID
	friendFUID := c.Param("fuid")
	if friendFUID == "" {
		fail(c, 400, "好友FUID不能为空")
		return
	}
	// 更新好友状态为正常
	err := db.Model(&Friend{}).Where("user_fuid = ? AND friend_fuid = ? AND status = 2",
		currentFUID, friendFUID).Update("status", 1).Error
	if err != nil {
		fail(c, 500, "移出黑名单失败: "+err.Error())
		return
	}
	success(c, map[string]string{"msg": "移出黑名单成功"})
	log.Infof("Remove blacklist: user=%s, friend=%s", currentFUID, friendFUID)
}

// 创建群聊接口
func createGroupHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 参数绑定
	var req struct {
		Name        string `json:"name" binding:"required,min=1,max=20"`
		Desc        string `json:"desc" binding:"max=256"`
		Avatar      string `json:"avatar"` // 头像URL
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: "+err.Error())
		return
	}
	// 检查创建群聊数量是否超限
	var groupCount int64
	db.Model(&Group{}).Where("owner_fuid = ? AND status = 1", currentFUID).Count(&groupCount)
	if groupCount >= int64(cfg.Business.Group.CreateMax) {
		fail(c, 400, fmt.Sprintf("创建群聊数量已达上限(%d)", cfg.Business.Group.CreateMax))
		return
	}
	// 生成QUID
	quid, err := generateQUID()
	if err != nil {
		fail(c, 500, "生成QUID失败: "+err.Error())
		return
	}
	// 创建群聊
	newGroup := Group{
		QUID:       quid,
		Name:       req.Name,
		OwnerFUID:  currentFUID,
		Avatar:     req.Avatar,
		Desc:       req.Desc,
		VIPLevel:   0,
		VIPExp:     0,
		Status:     1,
	}
	if err := db.Create(&newGroup).Error; err != nil {
		fail(c, 500, "创建群聊失败: "+err.Error())
		return
	}
	// 添加群主到群成员
	groupMember := GroupMember{
		GroupQUID:  quid,
		UserFUID:   currentFUID,
		Role:       1, // 群主
		Status:     1,
	}
	if err := db.Create(&groupMember).Error; err != nil {
		// 回滚群聊创建
		db.Delete(&newGroup)
		fail(c, 500, "添加群主到群成员失败: "+err.Error())
		return
	}
	// 返回群聊信息
	respData := map[string]interface{}{
		"quid":        newGroup.QUID,
		"name":        newGroup.Name,
		"owner_fuid":  newGroup.OwnerFUID,
		"avatar":      newGroup.Avatar,
		"desc":        newGroup.Desc,
		"vip_level":   newGroup.VIPLevel,
		"vip_exp":     newGroup.VIPExp,
	}
	success(c, respData)
	log.Infof("Create group: quid=%s, owner=%s, name=%s", quid, currentFUID, req.Name)
}

// 搜索群聊接口
func searchGroupHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 获取搜索参数
	keyword := c.Query("keyword")
	if keyword == "" {
		fail(c, 400, "搜索关键词不能为空")
		return
	}
	// 搜索群聊（QUID/群名）
	var groups []Group
	condition := "quid LIKE ? OR name LIKE ? AND status = 1"
	likeKeyword := "%" + keyword + "%"
	if err := db.Where(condition, likeKeyword, likeKeyword).
		Select("quid, name, owner_fuid, avatar, desc, vip_level, vip_exp").
		Find(&groups).Error; err != nil {
		fail(c, 500, "搜索群聊失败: "+err.Error())
		return
	}
	// 检查是否已加入群聊
	var result []map[string]interface{}
	for _, group := range groups {
		var member GroupMember
		err := db.Where("group_quid = ? AND user_fuid = ? AND status = 1", group.QUID, currentFUID).First(&member).Error
		isJoined := err == nil
		role := 0
		if isJoined {
			role = int(member.Role)
		}
		// 获取群主昵称
		var owner User
		db.Where("fuid = ?", group.OwnerFUID).Select("nickname").First(&owner)
		result = append(result, map[string]interface{}{
			"quid":        group.QUID,
			"name":        group.Name,
			"owner_fuid":  group.OwnerFUID,
			"owner_nickname": owner.Nickname,
			"avatar":      group.Avatar,
			"desc":        group.Desc,
			"vip_level":   group.VIPLevel,
			"vip_exp":     group.VIPExp,
			"is_joined":   isJoined,
			"role":        role,
		})
	}
	success(c, result, int64(len(result)))
}

// 加入群聊接口
func joinGroupHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 参数绑定
	var req struct {
		GroupQUID string `json:"group_quid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: "+err.Error())
		return
	}
	// 检查群聊是否存在
	var group Group
	if err := db.Where("quid = ? AND status = 1", req.GroupQUID).First(&group).Error; err != nil {
		fail(c, 400, "群聊不存在或已解散")
		return
	}
	// 检查是否已加入
	var member GroupMember
	err := db.Where("group_quid = ? AND user_fuid = ?", req.GroupQUID, currentFUID).First(&member).Error
	if err == nil {
		if member.Status == 1 {
			fail(c, 400, "已加入该群聊")
		} else if member.Status == 0 {
			// 恢复群成员身份
			member.Status = 1
			if err := db.Save(&member).Error; err != nil {
				fail(c, 500, "恢复群成员身份失败: "+err.Error())
				return
			}
			success(c, map[string]string{"msg": "重新加入群聊成功"})
			return
		}
	}
	// 检查群成员数量是否超限
	var memberCount int64
	db.Model(&GroupMember{}).Where("group_quid = ? AND status = 1", req.GroupQUID).Count(&memberCount)
	if memberCount >= int64(cfg.Business.Group.MemberMax) {
		fail(c, 400, fmt.Sprintf("群成员数量已达上限(%d)", cfg.Business.Group.MemberMax))
		return
	}
	// 加入群聊
	newMember := GroupMember{
		GroupQUID:  req.GroupQUID,
		UserFUID:   currentFUID,
		Role:       0, // 普通成员
		Status:     1,
	}
	if err := db.Create(&newMember).Error; err != nil {
		fail(c, 500, "加入群聊失败: "+err.Error())
		return
	}
	// 发送ntfy推送（如果启用）
	if cfg.Business.Notify.Ntfy.Enable {
		go sendNtfyNotification("入群通知", fmt.Sprintf("用户%s(%s)加入群聊%s",
			c.GetString("nickname"), currentFUID, group.Name), group.OwnerFUID)
	}
	success(c, map[string]string{"msg": "加入群聊成功"})
	log.Infof("Join group: user=%s, group=%s", currentFUID, req.GroupQUID)
}

// 退出群聊接口
func quitGroupHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 获取群聊QUID
	groupQUID := c.Param("quid")
	if groupQUID == "" {
		fail(c, 400, "群聊QUID不能为空")
		return
	}
	// 检查是否是群主
	var member GroupMember
	err := db.Where("group_quid = ? AND user_fuid = ? AND role = 1", groupQUID, currentFUID).First(&member).Error
	if err == nil {
		fail(c, 400, "群主不能退出群聊，请转让群主或解散群聊")
		return
	}
	// 更新群成员状态为已退出
	err = db.Model(&GroupMember{}).Where("group_quid = ? AND user_fuid = ? AND status = 1",
		groupQUID, currentFUID).Update("status", 0).Error
	if err != nil {
		fail(c, 500, "退出群聊失败: "+err.Error())
		return
	}
	success(c, map[string]string{"msg": "退出群聊成功"})
	log.Infof("Quit group: user=%s, group=%s", currentFUID, groupQUID)
}

// 群聊禁言接口
func groupMuteHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 参数绑定
	var req struct {
		GroupQUID   string `json:"group_quid" binding:"required"`
		UserFUID    string `json:"user_fuid" binding:"required"` // 被禁言用户
		MuteTime    int    `json:"mute_time"` // 禁言时间（秒），默认使用配置值
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: "+err.Error())
		return
	}
	// 检查当前用户权限（群主/管理）
	var currentMember GroupMember
	err := db.Where("group_quid = ? AND user_fuid = ? AND status = 1", req.GroupQUID, currentFUID).First(&currentMember).Error
	if err != nil {
		fail(c, 403, "你不是该群成员，无操作权限")
		return
	}
	if currentMember.Role != 1 && currentMember.Role != 2 {
		fail(c, 403, "仅群主和管理员可执行禁言操作")
		return
	}
	// 检查被禁言用户是否是群成员
	var targetMember GroupMember
	err = db.Where("group_quid = ? AND user_fuid = ? AND status = 1", req.GroupQUID, req.UserFUID).First(&targetMember).Error
	if err != nil {
		fail(c, 400, "被禁言用户不是该群成员")
		return
	}
	// 检查被禁言用户是否是群主
	if targetMember.Role == 1 {
		fail(c, 400, "不能禁言群主")
		return
	}
	// 计算禁言结束时间
	muteTime := req.MuteTime
	if muteTime <= 0 {
		muteTime = cfg.Business.Group.MuteTimeDefault
	}
	muteEndTime := time.Now().Add(time.Duration(muteTime) * time.Second)
	// 更新禁言时间
	err = db.Model(&targetMember).Update("mute_end_time", muteEndTime).Error
	if err != nil {
		fail(c, 500, "禁言操作失败: "+err.Error())
		return
	}
	// 发送ntfy推送
	if cfg.Business.Notify.Ntfy.Enable {
		go sendNtfyNotification("群禁言通知", fmt.Sprintf("你在群聊%s中被禁言%d秒",
			req.GroupQUID, muteTime), req.UserFUID)
	}
	success(c, map[string]string{"msg": fmt.Sprintf("禁言成功，禁言时长%d秒", muteTime)})
	log.Infof("Group mute: group=%s, operator=%s, target=%s, time=%d", req.GroupQUID, currentFUID, req.UserFUID, muteTime)
}

// 踢出群聊接口
func kickGroupMemberHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 参数绑定
	var req struct {
		GroupQUID   string `json:"group_quid" binding:"required"`
		UserFUID    string `json:"user_fuid" binding:"required"` // 被踢出用户
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: "+err.Error())
		return
	}
	// 检查当前用户权限（群主/管理）
	var currentMember GroupMember
	err := db.Where("group_quid = ? AND user_fuid = ? AND status = 1", req.GroupQUID, currentFUID).First(&currentMember).Error
	if err != nil {
		fail(c, 403, "你不是该群成员，无操作权限")
		return
	}
	if currentMember.Role != 1 && currentMember.Role != 2 {
		fail(c, 403, "仅群主和管理员可执行踢出操作")
		return
	}
	// 检查被踢出用户是否是群成员
	var targetMember GroupMember
	err = db.Where("group_quid = ? AND user_fuid = ? AND status = 1", req.GroupQUID, req.UserFUID).First(&targetMember).Error
	if err != nil {
		fail(c, 400, "被踢出用户不是该群成员")
		return
	}
	// 检查被踢出用户是否是群主
	if targetMember.Role == 1 {
		fail(c, 400, "不能踢出群主")
		return
	}
	// 更新群成员状态为已踢出
	err = db.Model(&targetMember).Update("status", 0).Error
	if err != nil {
		fail(c, 500, "踢出操作失败: "+err.Error())
		return
	}
	// 发送ntfy推送
	if cfg.Business.Notify.Ntfy.Enable {
		go sendNtfyNotification("踢出群聊通知", fmt.Sprintf("你被移出群聊%s", req.GroupQUID), req.UserFUID)
	}
	success(c, map[string]string{"msg": "踢出群聊成功"})
	log.Infof("Kick group member: group=%s, operator=%s, target=%s", req.GroupQUID, currentFUID, req.UserFUID)
}

// 发布群公告接口
func publishGroupNoticeHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 参数绑定
	var req struct {
		GroupQUID   string `json:"group_quid" binding:"required"`
		Content     string `json:"content" binding:"required,max=256"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: "+err.Error())
		return
	}
	// 检查当前用户权限（群主/管理）
	var currentMember GroupMember
	err := db.Where("group_quid = ? AND user_fuid = ? AND status = 1", req.GroupQUID, currentFUID).First(&currentMember).Error
	if err != nil {
		fail(c, 403, "你不是该群成员，无操作权限")
		return
	}
	if currentMember.Role != 1 && currentMember.Role != 2 {
		fail(c, 403, "仅群主和管理员可发布群公告")
		return
	}
	// 创建群公告
	notice := GroupNotice{
		GroupQUID:     req.GroupQUID,
		Content:       req.Content,
		PublisherFUID: currentFUID,
		PublishTime:   time.Now(),
	}
	if err := db.Create(&notice).Error; err != nil {
		fail(c, 500, "发布群公告失败: "+err.Error())
		return
	}
	// 推送群公告给所有成员
	go func() {
		var members []GroupMember
		db.Where("group_quid = ? AND status = 1", req.GroupQUID).Find(&members)
		for _, member := range members {
			if cfg.Business.Notify.Ntfy.Enable {
				sendNtfyNotification("群公告", fmt.Sprintf("群聊%s发布新公告: %s", req.GroupQUID, req.Content), member.UserFUID)
			}
		}
	}()
	success(c, map[string]string{"msg": "发布群公告成功"})
	log.Infof("Publish group notice: group=%s, publisher=%s, content=%s", req.GroupQUID, currentFUID, req.Content)
}

// 解散群聊接口
func dissolveGroupHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 获取群聊QUID
	groupQUID := c.Param("quid")
	if groupQUID == "" {
		fail(c, 400, "群聊QUID不能为空")
		return
	}
	// 检查是否是群主
	var group Group
	err := db.Where("quid = ? AND owner_fuid = ? AND status = 1", groupQUID, currentFUID).First(&group).Error
	if err != nil {
		fail(c, 403, "仅群主可解散群聊")
		return
	}
	// 更新群聊状态为已解散
	err = db.Model(&group).Update("status", 0).Error
	if err != nil {
		fail(c, 500, "解散群聊失败: "+err.Error())
		return
	}
	// 更新所有群成员状态为已退出
	db.Model(&GroupMember{}).Where("group_quid = ?", groupQUID).Update("status", 0)
	// 推送解散通知
	go func() {
		var members []GroupMember
		db.Where("group_quid = ? AND status = 1", groupQUID).Find(&members)
		for _, member := range members {
			if cfg.Business.Notify.Ntfy.Enable {
				sendNtfyNotification("群聊解散通知", fmt.Sprintf("群聊%s已被群主解散", groupQUID), member.UserFUID)
			}
		}
	}()
	success(c, map[string]string{"msg": "解散群聊成功"})
	log.Infof("Dissolve group: group=%s, owner=%s", groupQUID, currentFUID)
}

// 转让群聊接口
func transferGroupHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 参数绑定
	var req struct {
		GroupQUID   string `json:"group_quid" binding:"required"`
		TargetFUID  string `json:"target_fuid" binding:"required"` // 转让目标用户
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: "+err.Error())
		return
	}
	// 检查是否是群主
	var group Group
	err := db.Where("quid = ? AND owner_fuid = ? AND status = 1", req.GroupQUID, currentFUID).First(&group).Error
	if err != nil {
		fail(c, 403, "仅群主可转让群聊")
		return
	}
	// 检查目标用户是否是群成员
	var targetMember GroupMember
	err = db.Where("group_quid = ? AND user_fuid = ? AND status = 1", req.GroupQUID, req.TargetFUID).First(&targetMember).Error
	if err != nil {
		fail(c, 400, "转让目标用户不是该群成员")
		return
	}
	// 更新群主
	err = db.Transaction(func(tx *gorm.DB) error {
		// 更新群聊群主
		if err := tx.Model(&group).Update("owner_fuid", req.TargetFUID).Error; err != nil {
			return err
		}
		// 更新原群主角色为普通成员
		if err := tx.Model(&GroupMember{}).Where("group_quid = ? AND user_fuid = ?", req.GroupQUID, currentFUID).Update("role", 0).Error; err != nil {
			return err
		}
		// 更新新群主角色为群主
		if err := tx.Model(&targetMember).Update("role", 1).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		fail(c, 500, "转让群聊失败: "+err.Error())
		return
	}
	// 发送转让通知
	if cfg.Business.Notify.Ntfy.Enable {
		// 通知新群主
		go sendNtfyNotification("群聊转让通知", fmt.Sprintf("你已成为群聊%s的新群主", req.GroupQUID), req.TargetFUID)
		// 通知原群主
		go sendNtfyNotification("群聊转让通知", fmt.Sprintf("你已将群聊%s转让给用户%s", req.GroupQUID, req.TargetFUID), currentFUID)
	}
	success(c, map[string]string{"msg": "转让群聊成功"})
	log.Infof("Transfer group: group=%s, from=%s, to=%s", req.GroupQUID, currentFUID, req.TargetFUID)
}

// 发送消息接口
func sendMessageHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 参数绑定
	var req struct {
		ReceiverType uint8  `json:"receiver_type" binding:"required,oneof=1 2"` // 1:单聊 2:群聊
		ReceiverID   string `json:"receiver_id" binding:"required"` // 单聊:好友FUID 群聊:群QUID
		ContentType  uint8  `json:"content_type" binding:"required,oneof=1 2 3 4 5"` // 1:文字 2:图片 3:文件 4:表情 5:系统消息
		Content      string `json:"content" binding:"required"` // 加密后的内容
		FontStyle    string `json:"font_style"` // 字体样式
		FontSize     int    `json:"font_size"`  // 字体大小
		FontColor    string `json:"font_color"` // 字体颜色
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: "+err.Error())
		return
	}
	// 验证接收方合法性
	if req.ReceiverType == 1 {
		// 单聊：检查是否是好友且不在黑名单
		var friend Friend
		err := db.Where("user_fuid = ? AND friend_fuid = ? AND status = 1", currentFUID, req.ReceiverID).First(&friend).Error
		if err != nil {
			fail(c, 400, "该用户不是你的好友，无法发送消息")
			return
		}
		// 检查对方是否将自己加入黑名单
		var reverseFriend Friend
		err = db.Where("user_fuid = ? AND friend_fuid = ? AND status = 2", req.ReceiverID, currentFUID).First(&reverseFriend).Error
		if err == nil {
			fail(c, 403, "对方已将你加入黑名单，无法发送消息")
			return
		}
	} else if req.ReceiverType == 2 {
		// 群聊：检查是否是群成员，且未被禁言
		var member GroupMember
		err := db.Where("group_quid = ? AND user_fuid = ? AND status = 1", req.ReceiverID, currentFUID).First(&member).Error
		if err != nil {
			fail(c, 403, "你不是该群成员，无法发送消息")
			return
		}
		// 检查是否被禁言
		if member.MuteEndTime.After(time.Now()) {
			fail(c, 403, fmt.Sprintf("你已被禁言，禁言结束时间：%s", member.MuteEndTime.Format("2006-01-02 15:04:05")))
			return
		}
	}
	// 生成消息ID
	msgID, err := generateUniqueID(32)
	if err != nil {
		fail(c, 500, "生成消息ID失败: "+err.Error())
		return
	}
	// 处理字体参数默认值
	fontStyle := req.FontStyle
	if fontStyle == "" {
		fontStyle = "思源黑体"
	}
	fontSize := req.FontSize
	if fontSize <= 0 {
		fontSize = 14
	}
	fontColor := req.FontColor
	if fontColor == "" {
		fontColor = "#000000"
	}
	// 创建消息记录
	message := Message{
		MsgID:        msgID,
		SenderFUID:   currentFUID,
		ReceiverType: req.ReceiverType,
		ReceiverID:   req.ReceiverID,
		ContentType:  req.ContentType,
		Content:      req.Content, // 已加密内容
		FontStyle:    fontStyle,
		FontSize:     fontSize,
		FontColor:    fontColor,
		IsRecalled:   false,
		IsRead:       false,
		SendTime:     time.Now(),
	}
	if err := db.Create(&message).Error; err != nil {
		fail(c, 500, "保存消息失败: "+err.Error())
		return
	}
	// 处理离线消息
	go saveOfflineMessage(req.ReceiverType, req.ReceiverID, msgID)
	// 推送消息（socket.io）
	go pushMessageToClient(message)
	// 发送ntfy推送（离线时）
	go func() {
		var targetFUIDs []string
		if req.ReceiverType == 1 {
			targetFUIDs = append(targetFUIDs, req.ReceiverID)
		} else {
			// 群聊：获取所有在线成员（排除自己）
			var members []GroupMember
			db.Where("group_quid = ? AND user_fuid != ? AND status = 1", req.ReceiverID, currentFUID).Find(&members)
			for _, m := range members {
				targetFUIDs = append(targetFUIDs, m.UserFUID)
			}
		}
		// 解密内容用于推送
		decryptedContent, err := rsaDecrypt([]byte(req.Content))
		if err != nil {
			log.Error("解密消息内容失败: ", err)
			return
		}
		// 检查用户是否在线（Redis中存在则在线）
		ctx := context.Background()
		for _, fuid := range targetFUIDs {
			online, err := rdb.Exists(ctx, "online_user:"+fuid).Result()
			if err != nil || online == 0 {
				// 离线推送
				if cfg.Business.Notify.Ntfy.Enable {
					var title string
					if req.ReceiverType == 1 {
						title = "好友消息"
					} else {
						title = "群聊消息"
					}
					sendNtfyNotification(title, string(decryptedContent), fuid)
				}
			}
		}
	}()
	success(c, map[string]interface{}{
		"msg_id": message.MsgID,
		"send_time": message.SendTime.Format("2006-01-02 15:04:05"),
	})
	log.Infof("Send message: msg_id=%s, sender=%s, receiver_type=%d, receiver_id=%s",
		msgID, currentFUID, req.ReceiverType, req.ReceiverID)
}

// 撤回消息接口
func recallMessageHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 获取消息ID
	msgID := c.Param("msg_id")
	if msgID == "" {
		fail(c, 400, "消息ID不能为空")
		return
	}
	// 查询消息
	var message Message
	err := db.Where("msg_id = ? AND sender_fuid = ?", msgID, currentFUID).First(&message).Error
	if err != nil {
		fail(c, 400, "消息不存在或不是你发送的")
		return
	}
	// 检查是否超过撤回时间
	recallTimeout := time.Duration(cfg.Business.Message.RecallTimeout) * time.Second
	if time.Since(message.SendTime) > recallTimeout {
		fail(c, 400, fmt.Sprintf("消息超过%d分钟，无法撤回", cfg.Business.Message.RecallTimeout/60))
		return
	}
	// 更新消息为已撤回
	err = db.Model(&message).Update("is_recalled", true).Error
	if err != nil {
		fail(c, 500, "撤回消息失败: "+err.Error())
		return
	}
	// 推送撤回通知
	go pushRecallMessageToClient(message)
	success(c, map[string]string{"msg": "撤回消息成功"})
	log.Infof("Recall message: msg_id=%s, sender=%s", msgID, currentFUID)
}

// 保存离线消息
func saveOfflineMessage(receiverType uint8, receiverID string, msgID string) {
	ctx := context.Background()
	if receiverType == 1 {
		// 单聊：保存到接收方离线消息
		offlineMsg := OfflineMessage{
			UserFUID: receiverID,
			MsgID:    msgID,
			Status:   0,
		}
		db.Create(&offlineMsg)
		// Redis记录离线消息数
		rdb.Incr(ctx, "offline_msg_count:"+receiverID)
	} else {
		// 群聊：保存到所有离线成员
		var members []GroupMember
		db.Where("group_quid = ? AND status = 1", receiverID).Find(&members)
		for _, member := range members {
			// 检查是否在线
			online, err := rdb.Exists(ctx, "online_user:"+member.UserFUID).Result()
			if err != nil || online == 0 {
				offlineMsg := OfflineMessage{
					UserFUID: member.UserFUID,
					MsgID:    msgID,
					Status:   0,
				}
				db.Create(&offlineMsg)
				// Redis记录离线消息数
				rdb.Incr(ctx, "offline_msg_count:"+member.UserFUID)
			}
		}
	}
}

// 推送消息到客户端（socket.io）
func pushMessageToClient(message Message) {
	// 构建推送数据
	pushData := map[string]interface{}{
		"msg_id":         message.MsgID,
		"sender_fuid":    message.SenderFUID,
		"receiver_type":  message.ReceiverType,
		"receiver_id":    message.ReceiverID,
		"content_type":   message.ContentType,
		"content":        message.Content,
		"font_style":     message.FontStyle,
		"font_size":      message.FontSize,
		"font_color":     message.FontColor,
		"is_recalled":    message.IsRecalled,
		"send_time":      message.SendTime.Format("2006-01-02 15:04:05"),
	}
	// 获取发送者信息
	var sender User
	db.Where("fuid = ?", message.SenderFUID).Select("nickname, vip_level").First(&sender)
	pushData["sender_nickname"] = sender.Nickname
	pushData["sender_vip_level"] = sender.VIPLevel

	ctx := context.Background()
	if message.ReceiverType == 1 {
		// 单聊：推送给接收方
		groupID := "user:" + message.ReceiverID
		// 检查接收方是否在线
		online, err := rdb.Exists(ctx, "online_user:"+message.ReceiverID).Result()
		if err == nil && online > 0 {
			// 通过socket.io推送
			// socketServer是全局socket.io服务实例
			socketServer.BroadcastToRoom("", groupID, "new_message", pushData)
			// 更新已读状态
			db.Model(&message).Update("is_read", true)
		}
	} else {
		// 群聊：推送给所有在线成员
		groupID := "group:" + message.ReceiverID
		socketServer.BroadcastToRoom("", groupID, "new_message", pushData)
		// 更新已读状态（在线成员）
		var members []GroupMember
		db.Where("group_quid = ? AND status = 1", message.ReceiverID).Find(&members)
		for _, m := range members {
			online, err := rdb.Exists(ctx, "online_user:"+m.UserFUID).Result()
			if err == nil && online > 0 {
				// 简化处理：群消息已读状态按用户维度，此处省略
			}
		}
	}
}

// 推送撤回消息通知
func pushRecallMessageToClient(message Message) {
	pushData := map[string]interface{}{
		"msg_id":        message.MsgID,
		"is_recalled":   true,
		"recall_time":   time.Now().Format("2006-01-02 15:04:05"),
	}
	if message.ReceiverType == 1 {
		groupID := "user:" + message.ReceiverID
		socketServer.BroadcastToRoom("", groupID, "recall_message", pushData)
	} else {
		groupID := "group:" + message.ReceiverID
		socketServer.BroadcastToRoom("", groupID, "recall_message", pushData)
	}
}

// 文件/图片上传接口
func uploadFileHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 获取文件类型（image/file）
	fileType := c.Query("type")
	if fileType != "image" && fileType != "file" {
		fail(c, 400, "文件类型必须是image或file")
		return
	}
	// 获取上传文件
	file, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		fail(c, 400, "获取文件失败: "+err.Error())
		return
	}
	defer file.Close()
	// 检查文件大小
	maxSize := cfg.Storage.File.MaxSize
	if fileType == "image" {
		maxSize = cfg.Storage.Image.MaxSize
	}
	if fileHeader.Size > int64(maxSize) {
		fail(c, 400, fmt.Sprintf("文件大小超过限制（最大%dMB）", maxSize/1024/1024))
		return
	}
	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	ext = strings.TrimPrefix(ext, ".")
	allowTypes := cfg.Storage.File.AllowTypes
	if fileType == "image" {
		allowTypes = cfg.Storage.Image.AllowTypes
	}
	allow := false
	for _, t := range allowTypes {
		if ext == t {
			allow = true
			break
		}
	}
	if !allow {
		fail(c, 400, fmt.Sprintf("不支持的文件类型，允许的类型：%s", strings.Join(allowTypes, ",")))
		return
	}
	// 生成文件名
	fileName := fmt.Sprintf("%s_%d.%s", currentFUID, time.Now().UnixNano(), ext)
	var fileURL string
	// 存储文件
	if cfg.Storage.Type == "minio" {
		// MinIO存储
		ctx := context.Background()
		_, err := minioClient.PutObject(ctx, cfg.Storage.MinIO.Bucket, fileName, file, fileHeader.Size, minio.PutObjectOptions{
			ContentType: http.DetectContentType([]byte(ext)),
		})
		if err != nil {
			fail(c, 500, "MinIO上传失败: "+err.Error())
			return
		}
		fileURL = fmt.Sprintf("%s/%s/%s", cfg.Storage.MinIO.Domain, cfg.Storage.MinIO.Bucket, fileName)
	} else {
		// 本地存储
		// 创建存储目录
		uploadDir := filepath.Join(cfg.Storage.Local.Path, fileType)
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			fail(c, 500, "创建存储目录失败: "+err.Error())
			return
		}
		// 保存文件
		filePath := filepath.Join(uploadDir, fileName)
		outFile, err := os.Create(filePath)
		if err != nil {
			fail(c, 500, "创建文件失败: "+err.Error())
			return
		}
		defer outFile.Close()
		_, err = io.Copy(outFile, file)
		if err != nil {
			fail(c, 500, "保存文件失败: "+err.Error())
			return
		}
		fileURL = fmt.Sprintf("%s/%s/%s", cfg.Storage.Local.Domain, fileType, fileName)
	}
	// RSA加密文件URL
	encryptedURL, err := rsaEncrypt([]byte(fileURL))
	if err != nil {
		fail(c, 500, "加密文件URL失败: "+err.Error())
		return
	}
	success(c, map[string]interface{}{
		"file_url": encryptedURL, // 返回加密后的URL
		"file_name": fileHeader.Filename,
		"file_size": fileHeader.Size,
	})
	log.Infof("Upload file: user=%s, type=%s, name=%s, size=%d, url=%s",
		currentFUID, fileType, fileHeader.Filename, fileHeader.Size, fileURL)
}

// 获取好友资料卡接口
func getFriendProfileHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 获取好友FUID
	friendFUID := c.Param("fuid")
	if friendFUID == "" {
		fail(c, 400, "好友FUID不能为空")
		return
	}
	// 查询好友信息
	var user User
	err := db.Where("fuid = ? AND status = 1", friendFUID).
		Select("fuid, nickname, vip_level, vip_exp, signature").
		First(&user).Error
	if err != nil {
		fail(c, 400, "好友不存在或已被禁用")
		return
	}
	// 检查是否是好友
	var friend Friend
	err = db.Where("user_fuid = ? AND friend_fuid = ? AND status = 1", currentFUID, friendFUID).First(&friend).Error
	if err != nil {
		fail(c, 403, "该用户不是你的好友，无法查看资料卡")
		return
	}
	// 构建资料卡数据
	profile := map[string]interface{}{
		"fuid":        user.FUID,
		"nickname":    user.Nickname,
		"vip_level":   user.VIPLevel,
		"vip_exp":     user.VIPExp,
		"signature":   user.Signature,
	}
	success(c, profile)
}

// 获取群聊资料卡接口
func getGroupProfileHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 获取群聊QUID
	groupQUID := c.Param("quid")
	if groupQUID == "" {
		fail(c, 400, "群聊QUID不能为空")
		return
	}
	// 查询群聊信息
	var group Group
	err := db.Where("quid = ? AND status = 1", groupQUID).
		Select("quid, owner_fuid, name, vip_level, vip_exp, desc").
		First(&group).Error
	if err != nil {
		fail(c, 400, "群聊不存在或已解散")
		return
	}
	// 检查是否是群成员
	var member GroupMember
	err = db.Where("group_quid = ? AND user_fuid = ? AND status = 1", groupQUID, currentFUID).First(&member).Error
	if err != nil {
		fail(c, 403, "你不是该群成员，无法查看资料卡")
		return
	}
	// 获取群主信息
	var owner User
	db.Where("fuid = ?", group.OwnerFUID).Select("nickname").First(&owner)
	// 构建资料卡数据
	profile := map[string]interface{}{
		"quid":          group.QUID,
		"owner_fuid":    group.OwnerFUID,
		"owner_nickname": owner.Nickname,
		"group_name":    group.Name,
		"vip_level":     group.VIPLevel,
		"vip_exp":       group.VIPExp,
		"desc":          group.Desc,
	}
	success(c, profile)
}

// 获取离线消息接口
func getOfflineMessageHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	// 查询离线消息
	var offlineMsgs []OfflineMessage
	err := db.Where("user_fuid = ? AND status = 0", currentFUID).Find(&offlineMsgs).Error
	if err != nil {
		fail(c, 500, "查询离线消息失败: "+err.Error())
		return
	}
	// 获取消息详情
	var msgIDs []string
	for _, om := range offlineMsgs {
		msgIDs = append(msgIDs, om.MsgID)
	}
	var messages []Message
	if len(msgIDs) > 0 {
		db.Where("msg_id IN ?", msgIDs).Find(&messages)
	}
	// 构建返回数据
	var result []map[string]interface{}
	for _, msg := range messages {
		result = append(result, map[string]interface{}{
			"msg_id":         msg.MsgID,
			"sender_fuid":    msg.SenderFUID,
			"receiver_type":  msg.ReceiverType,
			"receiver_id":    msg.ReceiverID,
			"content_type":   msg.ContentType,
			"content":        msg.Content,
			"font_style":     msg.FontStyle,
			"font_size":      msg.FontSize,
			"font_color":     msg.FontColor,
			"is_recalled":    msg.IsRecalled,
			"send_time":      msg.SendTime.Format("2006-01-02 15:04:05"),
		})
	}
	// 更新离线消息状态为已推送
	db.Model(&OfflineMessage{}).Where("user_fuid = ?", currentFUID).Update("status", 1)
	// 清空离线消息数
	ctx := context.Background()
	rdb.Del(ctx, "offline_msg_count:"+currentFUID)
	success(c, result, int64(len(result)))
}

// 获取未读消息数接口
func getUnreadMessageCountHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}
	ctx := context.Background()
	// 获取离线消息数（好友+群聊）
	offlineCount, err := rdb.Get(ctx, "offline_msg_count:"+currentFUID).Int64()
	if err != nil {
		offlineCount = 0
	}
	// 区分好友和群聊消息数
	// 1. 好友未读消息数
	var friendUnread int64
	db.Table("messages m").
		Joins("LEFT JOIN friends f ON m.receiver_id = f.friend_fuid AND f.user_fuid = ?", currentFUID).
		Where("m.receiver_type = 1 AND m.receiver_id = ? AND m.is_read = 0 AND m.is_recalled = 0", currentFUID).
		Count(&friendUnread)
	// 2. 群聊未读消息数
	var groupUnread int64
	db.Table("messages m").
		Joins("LEFT JOIN group_members gm ON m.receiver_id = gm.group_quid AND gm.user_fuid = ?", currentFUID).
		Where("m.receiver_type = 2 AND gm.status = 1 AND m.is_recalled = 0").
		Count(&groupUnread)
	// 构建返回数据
	countData := map[string]interface{}{
		"total":   offlineCount + friendUnread + groupUnread,
		"friend":  friendUnread,
		"group":   groupUnread,
		"offline": offlineCount,
	}
	success(c, countData)
}

// ntfy消息推送
func sendNtfyNotification(title, content, targetFUID string) {
	// 获取目标用户信息
	var user User
	err := db.Where("fuid = ?", targetFUID).Select("nickname, email").First(&user).Error
	if err != nil {
		log.Error("获取推送用户信息失败: ", err)
		return
	}
	// 构建推送数据
	pushData := map[string]string{
		"topic":   cfg.Business.Notify.Ntfy.Topic + "_" + targetFUID,
		"title":   title,
		"message": content,
		"priority": "high",
		"tags":    "chat_im",
	}
	// 转换为JSON
	jsonData, err := json.Marshal(pushData)
	if err != nil {
		log.Error("序列化推送数据失败: ", err)
		return
	}
	// 发送POST请求
	req, err := http.NewRequest("POST", cfg.Business.Notify.Ntfy.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Error("创建推送请求失败: ", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Error("发送推送请求失败: ", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Error("推送请求返回错误: ", string(body))
	}
}

// VIP等级更新定时任务
func vipLevelUpdateTask() {
	ticker := time.NewTicker(time.Duration(cfg.Business.VIP.UpdateInterval) * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		log.Info("开始更新用户VIP等级")
		// 1. 更新用户VIP经验
		var users []User
		db.Where("vip_start_time IS NOT NULL AND status = 1").Find(&users)
		for _, user := range users {
			// 计算在线时长（简化：此处假设用户在线状态通过Redis记录）
			ctx := context.Background()
			online, err := rdb.Exists(ctx, "online_user:"+user.FUID).Result()
			if err != nil || online == 0 {
				continue
			}
			// 添加经验
			newExp := user.VIPExp + uint64(cfg.Business.VIP.ExpPerHour)
			newLevel := uint8(newExp / 100) // 每100经验升1级，可配置
			if newLevel > uint8(cfg.Business.VIP.LevelMax) {
				newLevel = uint8(cfg.Business.VIP.LevelMax)
				newExp = uint64(cfg.Business.VIP.LevelMax) * 100
			}
			// 更新
			db.Model(&user).Updates(map[string]interface{}{
				"vip_exp":   newExp,
				"vip_level": newLevel,
			})
		}
		// 2. 更新群聊VIP经验
		var groups []Group
		db.Where("vip_start_time IS NOT NULL AND status = 1").Find(&groups)
		for _, group := range groups {
			// 添加经验
			newExp := group.VIPExp + uint64(cfg.Business.GroupVIP.ExpPerHour)
			newLevel := uint8(newExp / 100)
			if newLevel > uint8(cfg.Business.GroupVIP.LevelMax) {
				newLevel = uint8(cfg.Business.GroupVIP.LevelMax)
				newExp = uint64(cfg.Business.GroupVIP.LevelMax) * 100
			}
			// 更新
			db.Model(&group).Updates(map[string]interface{}{
				"vip_exp":   newExp,
				"vip_level": newLevel,
			})
		}
		log.Info("完成VIP等级更新")
	}
}

// 消息自动清理定时任务
func messageAutoCleanTask() {
	if !cfg.Business.Message.AutoClean.Enable {
		return
	}
	// 每天凌晨执行
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		log.Info("开始自动清理过期消息")
		// 计算清理时间点
		cleanTime := time.Now().AddDate(0, 0, -cfg.Business.Message.AutoClean.Days)
		// 删除过期消息
		result := db.Where("send_time < ?", cleanTime).Delete(&Message{})
		if result.Error != nil {
			log.Error("清理过期消息失败: ", result.Error)
		} else {
			log.Infof("清理过期消息成功，共删除%d条", result.RowsAffected)
		}
		// 删除关联的离线消息
		db.Where("created_at < ?", cleanTime).Delete(&OfflineMessage{})
	}
}

// 发送语音消息接口（扩展现有消息类型）
func sendVoiceMessageHandler(c *gin.Context) {
	// 获取当前用户FUID
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}

	// 参数绑定
	var req struct {
		ReceiverType uint8  `json:"receiver_type" binding:"required,oneof=1 2"` // 1:单聊 2:群聊
		ReceiverID   string `json:"receiver_id" binding:"required"`             // 接收方ID
		Duration     int    `json:"duration" binding:"required,min=1"`          // 语音时长(秒)
		VoiceURL     string `json:"voice_url" binding:"required"`               // 加密后的语音文件URL
		FontStyle    string `json:"font_style"`                                 // 可选文字样式
		FontSize     int    `json:"font_size"`                                  // 可选文字大小
		FontColor    string `json:"font_color"`                                 // 可选文字颜色
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: "+err.Error())
		return
	}

	// 验证接收方合法性（复用单聊/群聊验证逻辑）
	if req.ReceiverType == 1 {
		// 单聊：检查好友关系
		var friend Friend
		if err := db.Where("user_fuid = ? AND friend_fuid = ? AND status = 1", currentFUID, req.ReceiverID).First(&friend).Error; err != nil {
			fail(c, 400, "该用户不是你的好友，无法发送语音消息")
			return
		}
		// 检查是否被对方拉黑
		var reverseFriend Friend
		if err := db.Where("user_fuid = ? AND friend_fuid = ? AND status = 2", req.ReceiverID, currentFUID).First(&reverseFriend).Error; err == nil {
			fail(c, 403, "对方已将你加入黑名单，无法发送消息")
			return
		}
	} else if req.ReceiverType == 2 {
		// 群聊：检查群成员身份及禁言状态
		var member GroupMember
		if err := db.Where("group_quid = ? AND user_fuid = ? AND status = 1", req.ReceiverID, currentFUID).First(&member).Error; err != nil {
			fail(c, 403, "你不是该群成员，无法发送语音消息")
			return
		}
		if member.MuteEndTime.After(time.Now()) {
			fail(c, 403, fmt.Sprintf("你已被禁言，禁言结束时间：%s", member.MuteEndTime.Format("2006-01-02 15:04:05")))
			return
		}
	}

	// 生成消息ID
	msgID, err := generateUniqueID(32)
	if err != nil {
		fail(c, 500, "生成消息ID失败: "+err.Error())
		return
	}

	// 处理字体参数默认值
	fontStyle := req.FontStyle
	if fontStyle == "" {
		fontStyle = "思源黑体"
	}
	fontSize := req.FontSize
	if fontSize <= 0 {
		fontSize = 14
	}
	fontColor := req.FontColor
	if fontColor == "" {
		fontColor = "#000000"
	}

	// 保存语音消息（content_type=6表示语音消息）
	message := Message{
		MsgID:        msgID,
		SenderFUID:   currentFUID,
		ReceiverType: req.ReceiverType,
		ReceiverID:   req.ReceiverID,
		ContentType:  6, // 新增：6=语音消息
		Content:      fmt.Sprintf(`{"url":"%s","duration":%d}`, req.VoiceURL, req.Duration), // 加密的JSON内容
		FontStyle:    fontStyle,
		FontSize:     fontSize,
		FontColor:    fontColor,
		IsRecalled:   false,
		IsRead:       false,
		SendTime:     time.Now(),
	}
	if err := db.Create(&message).Error; err != nil {
		fail(c, 500, "保存语音消息失败: "+err.Error())
		return
	}

	// 处理离线消息
	go saveOfflineMessage(req.ReceiverType, req.ReceiverID, msgID)
	// 推送消息给接收方
	go pushMessageToClient(message)

	success(c, map[string]interface{}{
		"msg_id":    message.MsgID,
		"send_time": message.SendTime.Format("2006-01-02 15:04:05"),
		"duration":  req.Duration,
	})
	log.Infof("Send voice message: msg_id=%s, sender=%s, receiver_type=%d, receiver_id=%s, duration=%ds",
		msgID, currentFUID, req.ReceiverType, req.ReceiverID, req.Duration)
}

// 发起语音/视频通话接口
func initCallHandler(c *gin.Context) {
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}

	// 参数绑定
	var req struct {
		ReceiverType uint8  `json:"receiver_type" binding:"required,oneof=1 2"` // 1:单聊 2:群聊
		ReceiverID   string `json:"receiver_id" binding:"required"`             // 接收方ID
		CallType     uint8  `json:"call_type" binding:"required,oneof=1 2"`     // 1:语音 2:视频
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: "+err.Error())
		return
	}

	// 验证接收方合法性
	if req.ReceiverType == 1 {
		// 单聊：检查好友关系
		var friend Friend
		if err := db.Where("user_fuid = ? AND friend_fuid = ? AND status = 1", currentFUID, req.ReceiverID).First(&friend).Error; err != nil {
			fail(c, 400, "该用户不是你的好友，无法发起通话")
			return
		}
	} else if req.ReceiverType == 2 {
		// 群聊：检查群成员身份
		var member GroupMember
		if err := db.Where("group_quid = ? AND user_fuid = ? AND status = 1", req.ReceiverID, currentFUID).First(&member).Error; err != nil {
			fail(c, 403, "你不是该群成员，无法发起群通话")
			return
		}
	}

	// 生成通话ID
	callID, err := generateUniqueID(32)
	if err != nil {
		fail(c, 500, "生成通话ID失败: "+err.Error())
		return
	}

	// 创建通话记录（初始状态：等待接听）
	call := Call{
		CallID:       callID,
		SenderFUID:   currentFUID,
		ReceiverType: req.ReceiverType,
		ReceiverID:   req.ReceiverID,
		CallType:     req.CallType,
		Status:       0, // 0:等待接听
		CreateTime:   time.Now(),
	}
	if err := db.Create(&call).Error; err != nil {
		fail(c, 500, "创建通话记录失败: "+err.Error())
		return
	}

	// 推送通话请求给接收方
	go pushCallNotification(call, "incoming")

	success(c, map[string]interface{}{
		"call_id":     call.CallID,
		"create_time": call.CreateTime.Format("2006-01-02 15:04:05"),
		"status":      "等待对方接听",
	})
	log.Infof("Init call: call_id=%s, sender=%s, type=%d, receiver_type=%d, receiver_id=%s",
		callID, currentFUID, req.CallType, req.ReceiverType, req.ReceiverID)
}

// 接听通话接口
func acceptCallHandler(c *gin.Context) {
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}

	// 参数绑定
	var req struct {
		CallID string `json:"call_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: "+err.Error())
		return
	}

	// 查询通话记录并验证权限
	var call Call
	if err := db.Where("call_id = ?", req.CallID).First(&call).Error; err != nil {
		fail(c, 400, "通话记录不存在")
		return
	}

	// 验证是否为通话接收方
	if call.ReceiverType == 1 && call.ReceiverID != currentFUID {
		fail(c, 403, "无权操作此通话")
		return
	}
	if call.ReceiverType == 2 {
		// 群聊需验证是否为群成员
		var member GroupMember
		if err := db.Where("group_quid = ? AND user_fuid = ? AND status = 1", call.ReceiverID, currentFUID).First(&member).Error; err != nil {
			fail(c, 403, "你不是该群成员，无权接听")
			return
		}
	}

	// 验证通话状态（必须为等待接听状态）
	if call.Status != 0 {
		fail(c, 400, "该通话状态不允许接听")
		return
	}

	// 更新通话状态为通话中
	startTime := time.Now()
	if err := db.Model(&call).Updates(map[string]interface{}{
		"status":     1, // 1:通话中
		"start_time": startTime,
	}).Error; err != nil {
		fail(c, 500, "更新通话状态失败: "+err.Error())
		return
	}

	// 推送接听通知给发起方
	go pushCallNotification(call, "accepted")

	success(c, map[string]interface{}{
		"call_id":    call.CallID,
		"start_time": startTime.Format("2006-01-02 15:04:05"),
		"status":     "通话中",
	})
	log.Infof("Accept call: call_id=%s, receiver=%s", req.CallID, currentFUID)
}

// 拒绝通话接口
func rejectCallHandler(c *gin.Context) {
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}

	// 参数绑定
	var req struct {
		CallID  string `json:"call_id" binding:"required"`
		Reason  string `json:"reason" binding:"max=100"` // 拒绝原因（可选）
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: "+err.Error())
		return
	}

	// 查询通话记录并验证权限
	var call Call
	if err := db.Where("call_id = ?", req.CallID).First(&call).Error; err != nil {
		fail(c, 400, "通话记录不存在")
		return
	}

	// 验证是否为通话接收方
	if call.ReceiverType == 1 && call.ReceiverID != currentFUID {
		fail(c, 403, "无权操作此通话")
		return
	}

	// 验证通话状态
	if call.Status != 0 {
		fail(c, 400, "该通话状态不允许拒绝")
		return
	}

	// 更新通话状态为已拒绝
	if err := db.Model(&call).Updates(map[string]interface{}{
		"status":    2, // 2:已拒绝
		"end_time":  time.Now(),
	}).Error; err != nil {
		fail(c, 500, "更新通话状态失败: "+err.Error())
		return
	}

	// 推送拒绝通知给发起方
	go pushCallNotification(call, "rejected", req.Reason)

	success(c, map[string]string{
		"msg": "已拒绝通话",
	})
	log.Infof("Reject call: call_id=%s, receiver=%s, reason=%s", req.CallID, currentFUID, req.Reason)
}

// 结束通话接口
func endCallHandler(c *gin.Context) {
	currentFUID := c.GetString("fuid")
	if currentFUID == "" {
		fail(c, 401, "未登录")
		return
	}

	// 参数绑定
	var req struct {
		CallID string `json:"call_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: "+err.Error())
		return
	}

	// 查询通话记录并验证权限（发起者或接收者均可结束）
	var call Call
	if err := db.Where("call_id = ?", req.CallID).First(&call).Error; err != nil {
		fail(c, 400, "通话记录不存在")
		return
	}

	// 验证操作权限
	isSender := call.SenderFUID == currentFUID
	isReceiver := (call.ReceiverType == 1 && call.ReceiverID == currentFUID) || 
	              (call.ReceiverType == 2 && isGroupMember(call.ReceiverID, currentFUID))
	if !isSender && !isReceiver {
		fail(c, 403, "无权结束此通话")
		return
	}

	// 验证通话状态（必须为通话中）
	if call.Status != 1 {
		fail(c, 400, "该通话状态不允许结束")
		return
	}

	// 计算通话时长
	endTime := time.Now()
	duration := int(endTime.Sub(call.StartTime).Seconds())

	// 更新通话状态为已结束
	if err := db.Model(&call).Updates(map[string]interface{}{
		"status":   3,        // 3:已结束
		"end_time": endTime,
		"duration": duration,
	}).Error; err != nil {
		fail(c, 500, "更新通话状态失败: "+err.Error())
		return
	}

	// 推送结束通知给相关方
	go pushCallNotification(call, "ended", fmt.Sprintf("通话时长: %d秒", duration))

	success(c, map[string]interface{}{
		"msg":      "通话已结束",
		"duration": duration,
		"end_time": endTime.Format("2006-01-02 15:04:05"),
	})
	log.Infof("End call: call_id=%s, operator=%s, duration=%ds", req.CallID, currentFUID, duration)
}

// 推送通话通知（内部使用）
func pushCallNotification(call Call, action string, extra ...string) {
	// 构建通知内容
	notification := map[string]interface{}{
		"call_id":      call.CallID,
		"sender_fuid":  call.SenderFUID,
		"receiver_type": call.ReceiverType,
		"receiver_id":  call.ReceiverID,
		"call_type":    call.CallType,
		"action":       action, // incoming/accepted/rejected/ended
		"timestamp":    time.Now().Unix(),
	}
	if len(extra) > 0 {
		notification["extra"] = extra[0]
	}

	// 推送目标：单聊推送给接收者，群聊推送给所有群成员
	var targetFUIDs []string
	if call.ReceiverType == 1 {
		targetFUIDs = []string{call.ReceiverID}
	} else {
		var members []GroupMember
		db.Where("group_quid = ? AND status = 1", call.ReceiverID).Find(&members)
		for _, m := range members {
			targetFUIDs = append(targetFUIDs, m.UserFUID)
		}
	}
}



// 检查是否为群成员（内部使用）
func isGroupMember(groupQUID, userFUID string) bool {
	var count int64
	db.Model(&GroupMember{}).Where("group_quid = ? AND user_fuid = ? AND status = 1", groupQUID, userFUID).Count(&count)
	return count > 0
}

// 日志分割定时任务
func logRotateTask() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		log.Info("开始日志分割")
		// 关闭当前日志文件
		// 重新创建日志文件
		logFileName := filepath.Join(cfg.Log.Path, time.Now().Format(cfg.Log.FileNameFormat)+".log")
		file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Error("创建新日志文件失败: ", err)
			continue
		}
		log.SetOutput(io.MultiWriter(os.Stdout, file))
		// 清理旧日志文件
		cleanOldLogs()
		log.Info("完成日志分割")
	}
}

// 清理旧日志文件
func cleanOldLogs() {
	files, err := os.ReadDir(cfg.Log.Path)
	if err != nil {
		log.Error("读取日志目录失败: ", err)
		return
	}
	// 计算保留时间
	keepTime := time.Now().AddDate(0, 0, -cfg.Log.MaxAge)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		// 检查文件名是否符合格式
		if !strings.HasPrefix(file.Name(), "log_") {
			continue
		}
		// 检查文件修改时间
		fileInfo, err := file.Info()
		if err != nil {
		log.Error("获取文件信息失败: ", err, " 文件名称: ", file.Name())
		continue
		}

		// 检查文件修改时间是否早于保留时间
		if fileInfo.ModTime().Before(keepTime) {
		filePath := filepath.Join(cfg.Log.Path, file.Name())
		if err := os.Remove(filePath); err != nil {
		log.Error("删除旧日志文件失败: ", err, " 文件路径: ", filePath)
		} else {
		log.Infof("删除旧日志文件: %s", filePath)
			}
		}
	}
}

// 初始化Socket.IO服务
func initSocketIO() (*socketio.Server, error) {
	// 创建socket.io服务器
	server := socketio.NewServer(&engineio.Options{
		Transports: []transport.Transport{
			&transportWs.Transport{
				CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				for _, allowed := range cfg.App.CORS.AllowOrigins {
				if origin == allowed || allowed == "*" {
				return true
				}
				}
				return false
				}, 
			},
		},
	})
	// 连接事件
	server.OnConnect("/", func(s socketio.Conn) error {
		// 验证Token
		u := s.URL()
		token := (&u).Query().Get("token")
		if token == "" {
			return errors.New("未提供Token")
		}
		// 解码Token
		encryptedPayload, err := base64.StdEncoding.DecodeString(token)
		if err != nil {
			return errors.New("Token无效")
		}
		// AES解密
		aesKey := []byte(cfg.Crypto.JWT.Secret)
		payloadBytes, err := aesDecrypt(encryptedPayload, aesKey)
		if err != nil {
			return errors.New("Token无效")
		}
		// 解析Payload
		var payload map[string]interface{}
		err = json.Unmarshal(payloadBytes, &payload)
		if err != nil {
			return errors.New("Token无效")
		}
		// 检查过期时间
		exp, ok := payload["exp"].(float64)
		if !ok || int64(exp) < time.Now().Unix() {
			return errors.New("Token已过期")
		}
		fuid := payload["fuid"].(string)
		// 记录用户在线状态
		ctx := context.Background()
		rdb.Set(ctx, "online_user:"+fuid, s.ID(), time.Duration(cfg.Crypto.JWT.Expire)*time.Second)
		// 加入用户房间
		s.Join("user:" + fuid)
		// 加入所有群聊房间
		var groupMembers []GroupMember
		db.Where("user_fuid = ? AND status = 1", fuid).Find(&groupMembers)
		for _, gm := range groupMembers {
			s.Join("group:" + gm.GroupQUID)
		}
		log.Infof("Socket.IO connect: fuid=%s, conn_id=%s", fuid, s.ID())
		return nil
	})
	// 断开连接事件
	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		// 获取用户FUID
		ctx := context.Background()
		var fuid string
		// 遍历在线用户，找到对应的FUID
		iter := rdb.Scan(ctx, 0, "online_user:*", 0).Iterator()
		for iter.Next(ctx) {
			key := iter.Val()
			val, _ := rdb.Get(ctx, key).Result()
			if val == s.ID() {
				fuid = strings.TrimPrefix(key, "online_user:")
				break
			}
		}
		if fuid != "" {
			// 删除在线状态
			rdb.Del(ctx, "online_user:"+fuid)
			log.Infof("Socket.IO disconnect: fuid=%s, conn_id=%s, reason=%s", fuid, s.ID(), reason)
		}
	})
	// 错误事件
	server.OnError("/", func(s socketio.Conn, err error) {
		log.Error("Socket.IO error: conn_id=%s, err=%v", s.ID(), err)
	})
	return server, nil
}

func main() {
	// 记录服务启动时间
	startTime = time.Now()
	// 加载配置
	err := loadConfig()
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	err = initLog()
	if err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化MySQL
	err = initMySQL()
	if err != nil {
		log.Fatalf("初始化MySQL失败: %v", err)
	}

	// 初始化Redis
	err = initRedis()
	if err != nil {
		log.Fatalf("初始化Redis失败: %v", err)
	}

	// 初始化MinIO
	err = initMinIO()
	if err != nil {
		log.Fatalf("初始化MinIO失败: %v", err)
	}

	// 初始化RSA
	err = initRSA()
	if err != nil {
		log.Fatalf("初始化RSA失败: %v", err)
	}

	// 初始化Socket.IO
	socketServer, err = initSocketIO()
	if err != nil {
		log.Fatalf("初始化Socket.IO失败: %v", err)
	}

	// 设置Gin模式
	if cfg.App.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// 创建Gin引擎
	r := gin.Default()

	// 配置CORS
	corsConfig := cors.Config{
		AllowOrigins:     cfg.App.CORS.AllowOrigins,
		AllowMethods:     cfg.App.CORS.AllowMethods,
		AllowHeaders:     cfg.App.CORS.AllowHeaders,
		AllowCredentials: cfg.App.CORS.AllowCredentials,
		MaxAge:           time.Duration(cfg.App.CORS.MaxAge) * time.Second,
	}
	r.Use(cors.New(corsConfig))

	// 初始化限流中间件
	limiters := initRateLimiters()

	// 公开接口
	publicGroup := r.Group("/api/v1/public")
	{
		publicGroup.POST("/register", limiters["register_login_ip"], limiters["register_login_user"], registerHandler)
		publicGroup.POST("/login", limiters["register_login_ip"], limiters["register_login_user"], loginHandler)
		// 健康检查接口
        publicGroup.GET("/health", limiters["register_login_ip"], healthHandler)
	}

	// 私有接口（需要登录）
	privateGroup := r.Group("/api/v1/private")
	privateGroup.Use(authMiddleware())
	{
		// 设备管理
		privateGroup.GET("/devices", getDevicesHandler)
		privateGroup.POST("/devices/kick", kickDeviceHandler)
		// 好友相关
		privateGroup.GET("/friend/search", searchFriendHandler)
		privateGroup.POST("/friend/add", addFriendHandler)
		privateGroup.DELETE("/friend/:fuid", deleteFriendHandler)
		privateGroup.PUT("/friend/remark", updateFriendRemarkHandler)
		privateGroup.POST("/friend/blacklist/add/:fuid", addBlacklistHandler)
		privateGroup.POST("/friend/blacklist/remove/:fuid", removeBlacklistHandler)
		privateGroup.GET("/friend/profile/:fuid", getFriendProfileHandler)

		// 群聊相关
		privateGroup.POST("/group/create", createGroupHandler)
		privateGroup.GET("/group/search", searchGroupHandler)
		privateGroup.POST("/group/join", joinGroupHandler)
		privateGroup.DELETE("/group/quit/:quid", quitGroupHandler)
		privateGroup.POST("/group/mute", groupMuteHandler)
		privateGroup.POST("/group/kick", kickGroupMemberHandler)
		privateGroup.POST("/group/notice/publish", publishGroupNoticeHandler)
		privateGroup.DELETE("/group/dissolve/:quid", dissolveGroupHandler)
		privateGroup.POST("/group/transfer", transferGroupHandler)
		privateGroup.GET("/group/profile/:quid", getGroupProfileHandler)

		// 消息相关
		privateGroup.POST("/message/send", limiters["message_conn"], limiters["group_user"], sendMessageHandler)
		privateGroup.POST("/message/recall/:msg_id", recallMessageHandler)
		privateGroup.GET("/message/offline", getOfflineMessageHandler)
		privateGroup.GET("/message/unread/count", getUnreadMessageCountHandler)
		
		privateGroup.POST("/message/voice", authMiddleware(), sendVoiceMessageHandler)
		privateGroup.POST("/call/init", authMiddleware(), initCallHandler)
		privateGroup.POST("/call/accept", authMiddleware(), acceptCallHandler)
		privateGroup.POST("/call/reject", authMiddleware(), rejectCallHandler)
		privateGroup.POST("/call/end", authMiddleware(), endCallHandler)

		// 文件上传
		privateGroup.POST("/upload", uploadFileHandler)
	}

	// 注册Socket.IO路由
	r.GET("/socket.io/*any", gin.WrapH(socketServer))
	r.POST("/socket.io/*any", gin.WrapH(socketServer))

	// 启动定时任务
	go vipLevelUpdateTask()
	go messageAutoCleanTask()
	go logRotateTask()

	// 处理系统信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 确定服务端口
	var port int
	if cfg.App.Mode == "debug" {
		port = cfg.App.Port.DevHTTP
	} else {
		port = cfg.App.Port.ProdHTTPS
	}

	// 启动HTTP/HTTPS服务
	go func() {
		if err := startServer(r, port); err != nil {
			log.Fatalf("启动HTTP服务失败: %v", err)
		}
	}()

	log.Infof("服务运行模式：%s", cfg.App.Mode,)

	// 等待退出信号
	<-quit
	log.Info("开始关闭服务")

	// 关闭资源
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
	_ = rdb.Close()
	// MinIO客户端无显式关闭方法
	socketServer.Close()

	log.Info("服务关闭成功")
}

// 启动服务器（整合完善的TLS配置）
func startServer(r *gin.Engine, port int) error {
	addr := fmt.Sprintf(":%d", port)

	if cfg.App.Mode == "release" {
		// 生产模式：使用HTTPS并配置安全的TLS参数
		cert, err := loadTLSCerts()
		if err != nil {
			log.Errorf("生产模式证书配置错误: %v，请检查证书文件后重试", err)
			return err
		}

		// 配置安全的TLS参数
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			},
			CurvePreferences: []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		}

		// 启动HTTPS服务器
		server := &http.Server{
			Addr:      addr,
			Handler:   r,
			TLSConfig: tlsConfig,
		}

		log.Infof("生成模式，https端口: %d", port)
		return server.ListenAndServeTLS("", "")
	} else {
		// 调试模式：使用HTTP
		log.Infof("调试模式，http端口: %d", port)
		return r.Run(addr)
	}
}