package volcengine

var ModelList = []string{
	// 豆包语言模型
	"Doubao-pro-128k",
	"Doubao-pro-32k",
	"Doubao-pro-4k",
	"Doubao-lite-128k",
	"Doubao-lite-32k",
	"Doubao-lite-4k",
	"Doubao-embedding",
	// 即梦（Seedream）图片生成模型
	"doubao-seedream-4-5-251128",      // 最新版，支持组图、多图融合，最小2K分辨率
	"doubao-seedream-3-0-t2i-250415",  // 文生图
	"doubao-seededit-3-0-i2i-250628",  // 图生图
}

var ChannelName = "volcengine"
