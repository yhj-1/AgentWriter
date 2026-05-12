package common

// Session 相关常量
const (
	UserLoginState = "userLoginState"
	AdminRole      = "admin"
	UserRole       = "user"
	VIPRole        = "vip"
)

// 密码相关常量
const (
	PasswordSalt      = "yupi"
	DefaultPassword   = "12345678"
	MinAccountLength  = 4
	MinPasswordLength = 8
)

// 分页相关常量
const (
	DefaultPageNum  = 1
	DefaultPageSize = 10
	MaxPageSize     = 100
)

// 配额相关常量
const (
	DefaultQuota = 5 // 新用户默认配额
)

// SSE 相关常量
const (
	SSETimeoutMS       = 30 * 60 * 1000 // SSE 连接超时时间（毫秒）：30分钟
	SSEReconnectTimeMS = 3000           // SSE 重连时间（毫秒）：3秒
)

// SSE 消息类型
const (
	SSEMsgAgent1Complete   = "AGENT1_COMPLETE"   // 智能体1完成（生成标题方案）
	SSEMsgTitlesGenerated  = "TITLES_GENERATED"  // 标题方案已生成（等待用户选择）
	SSEMsgAgent2Streaming  = "AGENT2_STREAMING"  // 智能体2流式输出（大纲）
	SSEMsgAgent2Complete   = "AGENT2_COMPLETE"   // 智能体2完成（生成大纲）
	SSEMsgOutlineGenerated = "OUTLINE_GENERATED" // 大纲已生成（等待用户编辑）
	SSEMsgAgent3Streaming  = "AGENT3_STREAMING"  // 智能体3流式输出（正文）
	SSEMsgAgent3Complete   = "AGENT3_COMPLETE"   // 智能体3完成（生成正文）
	SSEMsgAgent4Complete   = "AGENT4_COMPLETE"   // 智能体4完成（配图需求分析）
	SSEMsgImageComplete    = "IMAGE_COMPLETE"    // 单张配图完成
	SSEMsgAgent5Complete   = "AGENT5_COMPLETE"   // 智能体5完成（生成配图）
	SSEMsgMergeComplete    = "MERGE_COMPLETE"    // 图文合成完成
	SSEMsgAllComplete      = "ALL_COMPLETE"      // 全部完成
	SSEMsgError            = "ERROR"             // 错误消息
)

// Pexels 相关常量
const (
	PexelsAPIURL               = "https://api.pexels.com/v1/search"
	PexelsPerPage              = 1
	PexelsOrientationLandscape = "landscape"
)

// Picsum 相关常量
const (
	PicsumURLTemplate = "https://picsum.photos/800/600?random=%d"
)

// Bing 表情包相关常量
const (
	BingImageSearchURL = "https://cn.bing.com/images/async"
	EmojiPackSuffix    = "熊猫头表情包"
	BingMaxImages      = 30
)

// SVG 相关常量
const (
	SVGFilePrefix    = "svg-chart"
	SVGDefaultWidth  = 800
	SVGDefaultHeight = 600
)
