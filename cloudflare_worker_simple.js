// Cloudflare Worker - New API 反向代理
// 版本: v1.2 - 简化版（直接放行所有静态资源）

const CONFIG = {
  RAILWAY_URL: 'https://new-api-production-bf11.up.railway.app',
  ADMIN_SECRET_PATH: '/admin-secret-2025-2333'
};

addEventListener('fetch', event => {
  event.respondWith(handleRequest(event.request))
});

async function handleRequest(request) {
  const url = new URL(request.url);
  const path = url.pathname;

  // ===== 优先处理：静态资源直接转发 =====
  // 必须在其他判断之前，确保资源加载
  if (isStaticResource(path)) {
    return proxyToRailway(request, path);
  }

  // ===== 管理员访问路径 =====
  if (path.startsWith(CONFIG.ADMIN_SECRET_PATH)) {
    const realPath = path.replace(CONFIG.ADMIN_SECRET_PATH, '') || '/';
    return proxyToRailway(request, realPath);
  }

  // ===== API端点 =====
  if (path.startsWith('/v1/')) {
    return proxyToRailway(request, path);
  }

  // ===== API相关端点 =====
  if (path.startsWith('/api/')) {
    return proxyToRailway(request, path);
  }

  // ===== 阻止敏感路径直接访问 =====
  if (isSensitivePath(path)) {
    return new Response('Not Found', { status: 404 });
  }

  // ===== 默认空白页 =====
  return new Response('', {
    status: 200,
    headers: { 'Content-Type': 'text/html' }
  });
}

// 判断是否为静态资源
function isStaticResource(path) {
  // 更全面的静态资源匹配
  return path.startsWith('/assets/') ||
         path.endsWith('.js') ||
         path.endsWith('.css') ||
         path.endsWith('.png') ||
         path.endsWith('.jpg') ||
         path.endsWith('.jpeg') ||
         path.endsWith('.gif') ||
         path.endsWith('.svg') ||
         path.endsWith('.ico') ||
         path.endsWith('.woff') ||
         path.endsWith('.woff2') ||
         path.endsWith('.ttf') ||
         path.endsWith('.eot') ||
         path === '/logo.png' ||
         path === '/favicon.ico';
}

// 敏感路径
function isSensitivePath(path) {
  return path === '/login' ||
         path === '/register' ||
         path === '/console' ||
         path === '/setup' ||
         path.startsWith('/console/') ||
         path.startsWith('/reset') ||
         path.startsWith('/oauth/');
}

// 代理到Railway
async function proxyToRailway(request, path) {
  const url = new URL(request.url);
  const railwayUrl = CONFIG.RAILWAY_URL + path + url.search;

  const headers = new Headers(request.headers);

  // 确保Host正确
  headers.set('Host', new URL(CONFIG.RAILWAY_URL).host);

  const modifiedRequest = new Request(railwayUrl, {
    method: request.method,
    headers: headers,
    body: request.method !== 'GET' && request.method !== 'HEAD' ? request.body : null,
    redirect: 'follow'
  });

  try {
    const response = await fetch(modifiedRequest);

    // 创建新响应，保留所有原始头
    const newResponse = new Response(response.body, {
      status: response.status,
      statusText: response.statusText,
      headers: response.headers
    });

    // 只删除可能暴露后端信息的头
    newResponse.headers.delete('Server');
    newResponse.headers.delete('X-Powered-By');

    return newResponse;
  } catch (error) {
    return new Response('Proxy Error: ' + error.message, {
      status: 502,
      headers: { 'Content-Type': 'text/plain' }
    });
  }
}

/*
使用说明：
此版本的关键特点：
1. 静态资源（JS/CSS/图片等）直接转发，不做任何限制
2. 优先级：静态资源 > 管理路径 > API > 敏感路径阻止 > 空白页
3. 保留完整的响应头，确保MIME类型正确

访问方式：
- https://newapi.likele-zoom.workers.dev → 空白页
- https://newapi.likele-zoom.workers.dev/admin-secret-2025-2333/ → 管理面板
- https://newapi.likele-zoom.workers.dev/v1/models → API
*/