# Seedream 4.5 API
POST https://ark.cn-beijing.volces.com/api/v3/images/generations

## 请求参数 
### 请求体

model string  
本次请求使用模型的 Model ID 或推理接入点 (Endpoint ID)。

prompt string  
用于生成图像的提示词，支持中英文。（查看提示词指南：Seedream 4.0 、Seedream 3.0）
建议不超过300个汉字或600个英文单词。字数过多信息容易分散，模型可能因此忽略细节，只关注重点，造成图片缺失部分元素。

image string/array 
doubao-seedream-3.0-t2i 不支持该参数输入的图片信息，支持 URL 或 Base64 编码。其中，doubao-seedream-4.5、doubao-seedream-4.0 支持单图或多图输入（查看多图融合示例），doubao-seededit-3.0-i2i 仅支持单图输入。
图片URL：请确保图片URL可被访问。
Base64编码：请遵循此格式data:image/<图片格式>;base64,<Base64编码>。注意 <图片格式> 需小写，如 data:image/png;base64,<base64_image>。
说明
传入图片需要满足以下条件：
图片格式：jpeg、png（doubao-seedream-4.5、doubao-seedream-4.0 模型新增支持 webp、bmp、tiff、gif 格式new）
宽高比（宽/高）范围：
[1/16, 16] (适用模型：doubao-seedream-4.5、doubao-seedream-4.0）
[1/3, 3] (适用模型：doubao-seedream-3.0-t2i、doubao-seededit-3.0-i2i）
宽高长度（px） > 14
大小：不超过 10MB
总像素：不超过 6000x6000=36000000 px （对单张图宽度和高度的像素乘积限制，而不是对宽度或高度的单独值进行限制）
doubao-seedream-4.5、doubao-seedream-4.0 最多支持传入 14 张参考图。
size  string 
指定生成图像的尺寸信息，支持以下两种方式，不可混用。
方式 1 | 指定生成图像的分辨率，并在prompt中用自然语言描述图片宽高比、图片形状或图片用途，最终由模型判断生成图片的大小。
可选值：2K、4K
方式 2 | 指定生成图像的宽高像素值：
默认值：2048x2048
总像素取值范围：[2560x1440=3686400, 4096x4096=16777216] 
宽高比取值范围：[1/16, 16]
说明
采用方式 2 时，需同时满足总像素取值范围和宽高比取值范围。其中，总像素是对单张图宽度和高度的像素乘积限制，而不是对宽度或高度的单独值进行限制。
有效示例：3750x1250
总像素值 3750x1250=4687500，符合 [3686400, 16777216] 的区间要求；宽高比 3750/1250=3，符合 [1/16, 16] 的区间要求，故该示例值有效。
无效示例：1500x1500
总像素值 1500x1500=2250000，未达到 3686400 的最低要求；宽高 1500/1500=1，虽符合 [1/16, 16] 的区间要求，但因其未同时满足两项限制，故该示例值无效。
推荐的宽高像素值：
宽高比
宽高像素值
1:1
2048x2048
4:3
2304x1728
3:4
1728x2304
16:9
2560x1440
9:16
1440x2560
3:2
2496x1664
2:3
1664x2496
21:9
3024x1296
sequential_image_generation string 默认值 disabled
仅 doubao-seedream-4.5、doubao-seedream-4.0 支持该参数 | 查看组图输出示例控制是否关闭组图功能。
说明
组图：基于您输入的内容，生成的一组内容关联的图片。
auto：自动判断模式，模型会根据用户提供的提示词自主判断是否返回组图以及组图包含的图片数量。
disabled：关闭组图功能，模型只会生成一张图。
sequential_image_generation_options object
仅 doubao-seedream-4.5、doubao-seedream-4.0 支持该参数组图功能的配置。
stream  Boolean 默认值 false
仅 doubao-seedream-4.5、doubao-seedream-4.0 支持该参数 | 查看流式输出示例控制是否开启流式输出模式。
false：非流式输出模式，等待所有图片全部生成结束后再一次性返回所有信息。
true：流式输出模式，即时返回每张图片输出的结果。在生成单图和组图的场景下，流式输出模式均生效。
response_format string 默认值 url
指定生成图像的返回格式。
生成的图片为 jpeg 格式，支持以下两种返回方式：
url：返回图片下载链接；链接在图片生成后24小时内有效，请及时下载图片。
b64_json：以 Base64 编码字符串的 JSON 格式返回图像数据。
watermark  Boolean 默认值 true
是否在生成的图片中添加水印。
false：不添加水印。
true：在图片右下角添加“AI生成”字样的水印标识。
属性
optimize_prompt_options.mode string  默认值 standard
设置提示词优化功能使用的模式。
standard：标准模式，生成内容的质量更高，耗时较长。
fast：快速模式，生成内容的耗时更短，质量一般。
# 响应参数
流式响应参数
请参见文档。

非流式响应参数

model string
本次请求使用的模型 ID （模型名称-版本）。

created integer
本次请求创建时间的 Unix 时间戳（秒）。

data array
输出图像的信息。
说明
doubao-seedream-4.5、doubao-seedream-4.0 模型生成组图场景下，组图生成过程中某张图生成失败时：
若失败原因为审核不通过：仍会继续请求下一个图片生成任务，即不影响同请求内其他图片的生成流程。
若失败原因为内部服务异常（500）：不会继续请求下一个图片生成任务。
 

usage object
本次请求的用量信息。
 
error  object
本次请求，如发生错误，对应的错误信息。 
 
