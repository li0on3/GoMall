# GoMall - Go语言电商微服务平台

基于Go语言构建的电商网站前台微服务系统，采用现代微服务架构，提供完整的在线购物体验。

## 技术栈

- **后端框架**: Gin Web Framework
- **数据库**: MySQL 8.0 + Redis 6.0
- **ORM**: GORM v2
- **模板引擎**: Go HTML Template
- **API规范**: RESTful API + JSON
- **并发处理**: Goroutine + Channel
- **加密**: bcrypt + JWT

## 功能模块

### 用户管理
- 用户注册/登录
- JWT身份认证
- 用户信息管理

### 商品管理
- 商品展示和搜索
- 分类管理
- 文件上传
- Redis缓存优化

### 订单服务
- 购物车管理
- 订单处理
- 并发安全的库存管理

### API网关
- 统一路由管理
- 中间件集成
- HTML模板渲染

## 快速开始

### 环境要求
- Go 1.24+
- MySQL 8.0
- Redis 6.0

### 安装依赖
```bash
go mod tidy
```

### 运行应用
```bash
go run *.go
```

### API接口
- 用户注册: `POST /api/users/register`
- 用户登录: `POST /api/users/login`
- 商品列表: `GET /api/products`
- 购物车: `GET /api/cart`
- 创建订单: `POST /api/orders`

## 项目结构
```
GoMall/
├── main.go              # 主程序入口
├── config.go            # 配置管理
├── database.go          # 数据库连接
├── user.go             # 用户管理模块
├── product.go          # 商品管理模块
├── order.go            # 订单服务模块
├── api.go              # API路由
├── templates/          # HTML模板
├── public/             # 静态文件
└── upload/             # 上传文件
```

## 开发团队
- **用户管理模块**: 成员A
- **商品管理模块**: 成员B
- **订单服务模块**: 成员C
- **API网关和前端**: 成员D

## 许可证
MIT License