// Cloudflare Worker - 反向代理脚本
// 完全免费，无需VPS

const RAILWAY_URL = 'https://new-api-production-bf11.up.railway.app';
const ADMIN_SECRET_PATH = '/admin-secret-xyz-2025'; // 修改为您的秘密路径

addEventListener('fetch', event => {
  event.respondWith(handleRequest(event.request))
})

async function handleRequest(request) {
  const url = new URL(request.url);
  const path = url.pathname;

  // 🔐 管理员专用路径
  if (path.startsWith(ADMIN_SECRET_PATH)) {
    // 移除秘密路径前缀
    const realPath = path.replace(ADMIN_SECRET_PATH, '');
    url.pathname = realPath;

    // 转发到Railway
    const railwayRequest = new Request(RAILWAY_URL + url.pathname + url.search, {
      method: request.method,
      headers: request.headers,
      body: request.body,
      redirect: 'follow'
    });

    return fetch(railwayRequest);
  }

  // ✅ API端点保持公开
  if (path.startsWith('/v1/')) {
    const railwayRequest = new Request(RAILWAY_URL + path + url.search, {
      method: request.method,
      headers: request.headers,
      body: request.body,
      redirect: 'follow'
    });

    return fetch(railwayRequest);
  }

  // ❌ 阻止访问敏感路径
  if (path === '/login' ||
      path === '/register' ||
      path.startsWith('/console') ||
      path === '/setup') {
    return new Response('Not Found', { status: 404 });
  }

  // 🚫 默认返回空白页
  return new Response('', {
    status: 200,
    headers: {
      'Content-Type': 'text/html',
    }
  });
}

/*
部署步骤：

1. 登录 Cloudflare Dashboard (https://dash.cloudflare.com)
2. 进入 Workers & Pages
3. 创建 Worker
4. 粘贴此代码
5. 修改 ADMIN_SECRET_PATH 为您的秘密路径
6. 部署
7. 绑定自定义域名到Worker

访问方式：
- 普通用户: https://api.yourdomain.com → 空白页
- 管理员: https://api.yourdomain.com/admin-secret-xyz-2025/ → 管理面板
- API: https://api.yourdomain.com/v1/chat/completions → 正常工作

优势：
✅ 完全免费（Cloudflare免费套餐）
✅ 无需VPS
✅ 全球CDN加速
✅ 自动SSL证书
✅ 高可用性
✅ 零维护

限制：
- 免费版：10万次请求/天
- 如需更多，升级到$5/月（1000万次/天）
*/