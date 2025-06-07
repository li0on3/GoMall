#!/bin/bash

# 测试用户API脚本

echo "=== GoMall 用户管理API测试 ==="

# 1. 测试用户注册
echo "1. 测试用户注册..."
REGISTER_RESPONSE=$(curl -s -X POST http://localhost:8080/api/users/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "123456",
    "phone": "13800138000",
    "real_name": "测试用户"
  }')

echo "注册响应: $REGISTER_RESPONSE"

# 提取token
TOKEN=$(echo $REGISTER_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)
echo "获取到的Token: $TOKEN"

echo ""

# 2. 测试用户登录
echo "2. 测试用户登录..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/users/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "123456"
  }')

echo "登录响应: $LOGIN_RESPONSE"

# 如果注册失败，从登录响应中提取token
if [ -z "$TOKEN" ]; then
    TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)
    echo "从登录响应获取Token: $TOKEN"
fi

echo ""

# 3. 测试获取用户信息（需要认证）
echo "3. 测试获取用户信息..."
PROFILE_RESPONSE=$(curl -s -X GET http://localhost:8080/api/users/profile \
  -H "Authorization: Bearer $TOKEN")

echo "用户信息响应: $PROFILE_RESPONSE"

echo ""

# 4. 测试更新用户信息
echo "4. 测试更新用户信息..."
UPDATE_RESPONSE=$(curl -s -X PUT http://localhost:8080/api/users/profile \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "phone": "13900139000",
    "real_name": "更新后的用户名"
  }')

echo "更新响应: $UPDATE_RESPONSE"

echo ""

# 5. 测试修改密码
echo "5. 测试修改密码..."
PASSWORD_RESPONSE=$(curl -s -X PUT http://localhost:8080/api/users/password \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "old_password": "123456",
    "new_password": "654321"
  }')

echo "修改密码响应: $PASSWORD_RESPONSE"

echo ""

# 6. 测试新密码登录
echo "6. 测试新密码登录..."
NEW_LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/users/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "654321"
  }')

echo "新密码登录响应: $NEW_LOGIN_RESPONSE"

echo ""

# 7. 测试用户登出
echo "7. 测试用户登出..."
LOGOUT_RESPONSE=$(curl -s -X POST http://localhost:8080/api/users/logout \
  -H "Authorization: Bearer $TOKEN")

echo "登出响应: $LOGOUT_RESPONSE"

echo ""

# 8. 测试健康检查
echo "8. 测试健康检查..."
HEALTH_RESPONSE=$(curl -s http://localhost:8080/health)

echo "健康检查响应: $HEALTH_RESPONSE"

echo ""
echo "=== 用户API测试完成 ==="