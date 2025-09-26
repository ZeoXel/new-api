// Cloudflare Worker - New API 反向代理
// 版本: v1.0 - 优化版

// ========================================
// 🔧 配置区域 - 请修改以下参数
// ========================================

const CONFIG = {
  // Railway应用地址
  RAILWAY_URL: 'https://new-api-production-bf11.up.railway.app',

  // 🔐 管理员秘密路径（请修改为复杂的路径）
  // 示例: /admin-xyz-2025, /secret-panel-abc123
  ADMIN_SECRET_PATH: '/admin-secret-xyz-2025',

  // 是否启用调试日志（生产环境建议关闭）
  DEBUG: false
};

// ========================================
// 主处理函数
// ========================================

addEventListener('fetch', event => {
  event.respondWith(handleRequest(event.request))
});

async function handleRequest(request) {
  const url = new URL(request.url);
  const path = url.pathname;

  // 调试日志
  if (CONFIG.DEBUG) {
    console.log('Request Path:', path);
  }

  // 🔐 管理员访问路径
  if (path.startsWith(CONFIG.ADMIN_SECRET_PATH)) {
    return handleAdminAccess(request, url, path);
  }

  // ✅ API端点（公开访问）
  if (path.startsWith('/v1/')) {
    return handleAPIRequest(request, url, path);
  }

  // ❌ 阻止敏感路径
  if (isSensitivePath(path)) {
    return new Response('Not Found', {
      status: 404,
      headers: {
        'Content-Type': 'text/plain'
      }
    });
  }

  // 🚫 默认返回空白页
  return new Response('', {
    status: 200,
    headers: {
      'Content-Type': 'text/html',
      'Cache-Control': 'public, max-age=3600'
    }
  });
}

// ========================================
// 管理员访问处理
// ========================================

async function handleAdminAccess(request, url, path) {
  // 移除秘密路径前缀，获取真实路径
  const realPath = path.replace(CONFIG.ADMIN_SECRET_PATH, '') || '/';

  // 构建Railway URL
  const railwayUrl = new URL(CONFIG.RAILWAY_URL + realPath + url.search);

  // 创建新请求
  const modifiedRequest = new Request(railwayUrl, {
    method: request.method,
    headers: createProxyHeaders(request),
    body: request.method !== 'GET' && request.method !== 'HEAD' ? request.body : null,
    redirect: 'follow'
  });

  try {
    const response = await fetch(modifiedRequest);
    return modifyResponse(response);
  } catch (error) {
    return new Response('Proxy Error: ' + error.message, {
      status: 502,
      headers: { 'Content-Type': 'text/plain' }
    });
  }
}

// ========================================
// API请求处理
// ========================================

async function handleAPIRequest(request, url, path) {
  // 直接转发到Railway
  const railwayUrl = new URL(CONFIG.RAILWAY_URL + path + url.search);

  const modifiedRequest = new Request(railwayUrl, {
    method: request.method,
    headers: createProxyHeaders(request),
    body: request.method !== 'GET' && request.method !== 'HEAD' ? request.body : null,
    redirect: 'follow'
  });

  try {
    const response = await fetch(modifiedRequest);
    return modifyResponse(response);
  } catch (error) {
    return new Response('API Error: ' + error.message, {
      status: 502,
      headers: { 'Content-Type': 'application/json' }
    });
  }
}

// ========================================
// 辅助函数
// ========================================

// 检查是否为敏感路径
function isSensitivePath(path) {
  const sensitivePaths = [
    '/login',
    '/register',
    '/console',
    '/setup',
    '/reset',
    '/oauth'
  ];

  return sensitivePaths.some(sensitive =>
    path === sensitive || path.startsWith(sensitive + '/')
  );
}

// 创建代理请求头
function createProxyHeaders(request) {
  const headers = new Headers(request.headers);

  // 保留重要的请求头
  const importantHeaders = [
    'Authorization',
    'Content-Type',
    'User-Agent',
    'Accept',
    'Accept-Encoding',
    'Accept-Language'
  ];

  const newHeaders = new Headers();
  importantHeaders.forEach(header => {
    const value = headers.get(header);
    if (value) {
      newHeaders.set(header, value);
    }
  });

  // 添加代理信息
  newHeaders.set('X-Forwarded-For', request.headers.get('CF-Connecting-IP') || '');
  newHeaders.set('X-Real-IP', request.headers.get('CF-Connecting-IP') || '');

  return newHeaders;
}

// 修改响应
function modifyResponse(response) {
  const modifiedResponse = new Response(response.body, response);

  // 移除可能暴露后端信息的响应头
  modifiedResponse.headers.delete('Server');
  modifiedResponse.headers.delete('X-Powered-By');

  // 添加安全头
  modifiedResponse.headers.set('X-Content-Type-Options', 'nosniff');

  return modifiedResponse;
}

// ========================================
// 使用说明
// ========================================

/*
部署后的访问方式：

1. 普通用户访问（显示空白）：
   https://your-worker.workers.dev

2. 管理员访问（进入管理面板）：
   https://your-worker.workers.dev/admin-secret-xyz-2025/
   https://your-worker.workers.dev/admin-secret-xyz-2025/login

3. API调用（正常工作）：
   https://your-worker.workers.dev/v1/chat/completions
   https://your-worker.workers.dev/v1/models

注意事项：
- 必须修改 ADMIN_SECRET_PATH 为您的自定义路径
- 建议使用复杂路径，如：/admin-abc123-xyz789
- 可以绑定自定义域名以隐藏 .workers.dev 后缀
*/