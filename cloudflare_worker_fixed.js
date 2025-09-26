// Cloudflare Worker - New API 反向代理
// 版本: v1.1 - 修复版（解决静态资源加载问题）

// ========================================
// 🔧 配置区域
// ========================================

const CONFIG = {
  // Railway应用地址
  RAILWAY_URL: 'https://new-api-production-bf11.up.railway.app',

  // 🔐 管理员秘密路径（您已设置为 /admin-secret-2025-2333）
  ADMIN_SECRET_PATH: '/admin-secret-2025-2333',

  // 是否启用调试日志
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

  if (CONFIG.DEBUG) {
    console.log('Request Path:', path);
  }

  // 🔐 管理员访问路径及其静态资源
  if (path.startsWith(CONFIG.ADMIN_SECRET_PATH)) {
    return handleAdminAccess(request, url, path);
  }

  // ✅ 静态资源路径（/assets, /logo.png等）
  // 当用户已经通过秘密路径访问后，需要加载这些资源
  if (isStaticResource(path)) {
    // 检查referer，只允许从秘密路径访问的页面加载资源
    const referer = request.headers.get('Referer');
    if (referer && referer.includes(CONFIG.ADMIN_SECRET_PATH)) {
      return handleStaticResource(request, url, path);
    }
    // 否则返回空白，防止资源被盗链
    return new Response('', { status: 404 });
  }

  // ✅ API端点（公开访问）
  if (path.startsWith('/v1/')) {
    return handleAPIRequest(request, url, path);
  }

  // ✅ API状态端点
  if (path === '/api/status' || path.startsWith('/api/')) {
    const referer = request.headers.get('Referer');
    if (referer && referer.includes(CONFIG.ADMIN_SECRET_PATH)) {
      return proxyToRailway(request, url, path);
    }
  }

  // ❌ 阻止敏感路径直接访问
  if (isSensitivePath(path)) {
    return new Response('Not Found', { status: 404 });
  }

  // 🚫 默认返回空白页
  return new Response('', {
    status: 200,
    headers: { 'Content-Type': 'text/html' }
  });
}

// ========================================
// 判断是否为静态资源
// ========================================

function isStaticResource(path) {
  const staticPatterns = [
    '/assets/',
    '/logo.png',
    '/favicon.ico',
    '.js',
    '.css',
    '.png',
    '.jpg',
    '.svg',
    '.woff',
    '.woff2',
    '.ttf'
  ];

  return staticPatterns.some(pattern => path.includes(pattern));
}

// ========================================
// 管理员访问处理
// ========================================

async function handleAdminAccess(request, url, path) {
  // 移除秘密路径前缀
  let realPath = path.replace(CONFIG.ADMIN_SECRET_PATH, '');

  // 如果路径为空或只是/，重定向到首页
  if (!realPath || realPath === '/') {
    realPath = '/';
  }

  return proxyToRailway(request, url, realPath);
}

// ========================================
// 静态资源处理
// ========================================

async function handleStaticResource(request, url, path) {
  return proxyToRailway(request, url, path);
}

// ========================================
// API请求处理
// ========================================

async function handleAPIRequest(request, url, path) {
  return proxyToRailway(request, url, path);
}

// ========================================
// 代理到Railway
// ========================================

async function proxyToRailway(request, url, path) {
  const railwayUrl = new URL(CONFIG.RAILWAY_URL + path + url.search);

  const modifiedRequest = new Request(railwayUrl, {
    method: request.method,
    headers: createProxyHeaders(request),
    body: request.method !== 'GET' && request.method !== 'HEAD' ? request.body : null,
    redirect: 'follow'
  });

  try {
    const response = await fetch(modifiedRequest);
    return modifyResponse(response, path);
  } catch (error) {
    return new Response('Proxy Error: ' + error.message, {
      status: 502,
      headers: { 'Content-Type': 'text/plain' }
    });
  }
}

// ========================================
// 辅助函数
// ========================================

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

function createProxyHeaders(request) {
  const headers = new Headers(request.headers);

  const importantHeaders = [
    'Authorization',
    'Content-Type',
    'User-Agent',
    'Accept',
    'Accept-Encoding',
    'Accept-Language',
    'Cookie',
    'Referer'
  ];

  const newHeaders = new Headers();
  importantHeaders.forEach(header => {
    const value = headers.get(header);
    if (value) {
      newHeaders.set(header, value);
    }
  });

  newHeaders.set('X-Forwarded-For', request.headers.get('CF-Connecting-IP') || '');
  newHeaders.set('X-Real-IP', request.headers.get('CF-Connecting-IP') || '');

  return newHeaders;
}

function modifyResponse(response, path) {
  const modifiedResponse = new Response(response.body, response);

  // 移除后端信息
  modifiedResponse.headers.delete('Server');
  modifiedResponse.headers.delete('X-Powered-By');

  // 添加安全头
  modifiedResponse.headers.set('X-Content-Type-Options', 'nosniff');

  // 对于HTML页面，设置CSP允许加载资源
  if (response.headers.get('Content-Type')?.includes('text/html')) {
    modifiedResponse.headers.set('X-Frame-Options', 'SAMEORIGIN');
  }

  return modifiedResponse;
}

// ========================================
// 使用说明
// ========================================

/*
✅ 修复内容：
1. 添加静态资源路径检测（/assets/, /logo.png等）
2. 基于Referer判断，允许从管理路径加载的页面请求资源
3. 支持所有前端资源（JS, CSS, 图片, 字体等）
4. 保留Cookie和Referer头，确保会话正常

访问方式：
- 普通用户: https://newapi.likele-zoom.workers.dev → 空白页
- 管理员: https://newapi.likele-zoom.workers.dev/admin-secret-2025-2333/ → 管理面板 ✅
- API: https://newapi.likele-zoom.workers.dev/v1/models → 正常工作
*/