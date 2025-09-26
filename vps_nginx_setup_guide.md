# VPS + Nginx 反向代理完整部署指南

## 📋 前置要求

1. **一台VPS服务器**（推荐配置）
   - CPU: 1核
   - 内存: 1GB
   - 系统: Ubuntu 22.04 / Debian 11
   - 月费用: $3-5 (如Vultr、DigitalOcean、Linode)

2. **域名**
   - 已注册的域名（如 yourdomain.com）
   - 可配置DNS

3. **Railway应用**
   - 已部署的New API应用
   - URL: https://new-api-production-bf11.up.railway.app

---

## 🚀 Step 1: VPS基础配置

### 1.1 连接VPS
```bash
ssh root@your-vps-ip
```

### 1.2 更新系统
```bash
apt update && apt upgrade -y
```

### 1.3 安装必要软件
```bash
apt install -y nginx certbot python3-certbot-nginx ufw
```

### 1.4 配置防火墙
```bash
# 允许SSH、HTTP、HTTPS
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
```

---

## 🔧 Step 2: Nginx配置

### 2.1 创建配置文件
```bash
nano /etc/nginx/sites-available/api-proxy
```

### 2.2 粘贴以下配置
```nginx
# 反向代理配置
upstream railway_backend {
    server new-api-production-bf11.up.railway.app:443;
}

server {
    listen 80;
    server_name api.yourdomain.com;

    # 自动重定向到HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.yourdomain.com;

    # SSL证书（Step 3自动生成）
    ssl_certificate /etc/letsencrypt/live/api.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.yourdomain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # 隐藏服务器信息
    server_tokens off;
    more_clear_headers Server;
    more_clear_headers X-Powered-By;

    # 日志
    access_log /var/log/nginx/api_access.log;
    error_log /var/log/nginx/api_error.log;

    # 默认路径 - 返回空白页面
    location / {
        return 200 '';
        add_header Content-Type text/html;
    }

    # 🔐 管理员专用入口（修改为您的秘密路径）
    location /admin-xyz-secret-2025/ {
        # 移除秘密路径前缀，转发到真实后端
        rewrite ^/admin-xyz-secret-2025/(.*) /$1 break;

        proxy_pass https://railway_backend;
        proxy_ssl_server_name on;
        proxy_ssl_name new-api-production-bf11.up.railway.app;

        proxy_set_header Host new-api-production-bf11.up.railway.app;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket支持
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # 超时设置
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # ✅ API端点（保持公开，供实际调用）
    location /v1/ {
        proxy_pass https://railway_backend/v1/;
        proxy_ssl_server_name on;
        proxy_ssl_name new-api-production-bf11.up.railway.app;

        proxy_set_header Host new-api-production-bf11.up.railway.app;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header Authorization $http_authorization;

        # 超时设置（API可能需要更长时间）
        proxy_connect_timeout 300s;
        proxy_send_timeout 300s;
        proxy_read_timeout 300s;
    }

    # 阻止直接访问敏感路径
    location /login {
        return 403;
    }

    location /register {
        return 403;
    }

    location /console {
        return 403;
    }
}
```

### 2.3 启用配置
```bash
# 创建符号链接
ln -s /etc/nginx/sites-available/api-proxy /etc/nginx/sites-enabled/

# 删除默认配置
rm /etc/nginx/sites-enabled/default

# 测试配置
nginx -t

# 如果测试通过，重载Nginx
systemctl reload nginx
```

---

## 🔒 Step 3: 配置SSL证书

### 3.1 配置DNS
在您的域名提供商处添加A记录：
```
类型: A
主机: api
值: your-vps-ip
TTL: 300
```

等待5-10分钟让DNS生效，测试：
```bash
ping api.yourdomain.com
```

### 3.2 生成Let's Encrypt证书
```bash
certbot --nginx -d api.yourdomain.com
```

按提示输入：
- 邮箱地址
- 同意服务条款 (Y)
- 是否分享邮箱 (N)

### 3.3 自动续期
```bash
# 测试自动续期
certbot renew --dry-run

# 设置定时任务
crontab -e

# 添加这一行（每天凌晨2点检查续期）
0 2 * * * certbot renew --quiet
```

---

## ✅ Step 4: 验证部署

### 4.1 测试访问
```bash
# 普通访问 - 应该返回空白
curl https://api.yourdomain.com

# 管理访问 - 应该返回管理页面
curl https://api.yourdomain.com/admin-xyz-secret-2025/

# API调用 - 应该正常工作
curl https://api.yourdomain.com/v1/models \
  -H "Authorization: Bearer your-token"
```

### 4.2 浏览器测试
- 普通用户访问：`https://api.yourdomain.com` → 空白页 ✅
- 管理员访问：`https://api.yourdomain.com/admin-xyz-secret-2025/` → 登录页 ✅
- API调用：正常使用token调用 ✅

---

## 🔧 故障排查

### Nginx无法启动
```bash
# 查看错误日志
tail -f /var/log/nginx/error.log

# 检查配置语法
nginx -t

# 检查端口占用
netstat -tlnp | grep :80
netstat -tlnp | grep :443
```

### SSL证书问题
```bash
# 查看证书状态
certbot certificates

# 强制续期
certbot renew --force-renewal
```

### Railway连接失败
```bash
# 测试Railway连接
curl -I https://new-api-production-bf11.up.railway.app

# 如果失败，检查Railway服务是否正常运行
```

---

## 📊 架构总结

```
用户访问路径：

1. 普通用户
   https://api.yourdomain.com
   → Nginx (VPS)
   → 返回空白页 ❌ (隐藏管理面板)

2. 管理员
   https://api.yourdomain.com/admin-xyz-secret-2025/
   → Nginx (VPS)
   → Railway (New API) ✅
   → 管理面板

3. API调用
   https://api.yourdomain.com/v1/chat/completions
   → Nginx (VPS)
   → Railway (New API) ✅
   → 返回AI响应
```

---

## 💰 成本估算

| 项目 | 月费用 |
|------|--------|
| VPS (1核1GB) | $3-5 |
| 域名 | $1-2/月 |
| SSL证书 | 免费 (Let's Encrypt) |
| **总计** | **约$5/月** |

---

## 🎯 安全建议

1. **更改秘密路径**：
   ```nginx
   # 不要使用示例路径，自定义一个复杂的
   location /your-unique-secret-path-2025/
   ```

2. **IP白名单**（可选）：
   ```nginx
   location /admin-xyz-secret-2025/ {
       allow 1.2.3.4;  # 您的IP
       deny all;
       # ... 其他配置
   }
   ```

3. **速率限制**：
   ```nginx
   limit_req_zone $binary_remote_addr zone=admin:10m rate=5r/m;

   location /admin-xyz-secret-2025/ {
       limit_req zone=admin burst=10;
       # ... 其他配置
   }
   ```

4. **定期更新**：
   ```bash
   apt update && apt upgrade -y
   ```

---

## 🎉 完成！

现在您的New API网关已经：
- ✅ 完全隐藏管理面板
- ✅ 只有管理员知道访问路径
- ✅ API功能正常工作
- ✅ SSL加密保护
- ✅ Railway真实地址完全隐藏