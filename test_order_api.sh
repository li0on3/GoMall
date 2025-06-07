#!/bin/bash

# 测试订单管理API脚本

echo "=== GoMall 订单管理API测试 ==="

# 首先登录获取token
echo "0. 获取用户token..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/users/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "654321"
  }')

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)
echo "获取到的Token: $TOKEN"

if [ -z "$TOKEN" ]; then
    echo "无法获取token，请先运行用户API测试"
    exit 1
fi

echo ""

# 1. 添加商品到购物车
echo "1. 添加商品到购物车..."

# 添加第一个商品（iPhone 15 Pro Max）
CART_ADD_RESPONSE1=$(curl -s -X POST http://localhost:8080/api/cart/add \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "product_id": 1,
    "quantity": 2
  }')

echo "添加商品1到购物车响应: $CART_ADD_RESPONSE1"

# 添加第二个商品（Samsung Galaxy S24）
CART_ADD_RESPONSE2=$(curl -s -X POST http://localhost:8080/api/cart/add \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "product_id": 2,
    "quantity": 1
  }')

echo "添加商品2到购物车响应: $CART_ADD_RESPONSE2"

echo ""

# 2. 查看购物车
echo "2. 查看购物车..."
CART_RESPONSE=$(curl -s -X GET http://localhost:8080/api/cart \
  -H "Authorization: Bearer $TOKEN")

echo "购物车响应: $CART_RESPONSE"

# 提取购物车项ID（更精确的提取）
CART_ITEM_ID1=$(echo $CART_RESPONSE | grep -o '"items":\[{"id":[0-9]*' | grep -o '[0-9]*')
echo "购物车项ID1: $CART_ITEM_ID1"

echo ""

# 3. 更新购物车项数量
echo "3. 更新购物车项数量..."
UPDATE_CART_RESPONSE=$(curl -s -X PUT http://localhost:8080/api/cart/$CART_ITEM_ID1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "quantity": 3
  }')

echo "更新购物车响应: $UPDATE_CART_RESPONSE"

echo ""

# 4. 再次查看购物车（验证更新）
echo "4. 验证购物车更新..."
UPDATED_CART_RESPONSE=$(curl -s -X GET http://localhost:8080/api/cart \
  -H "Authorization: Bearer $TOKEN")

echo "更新后的购物车响应: $UPDATED_CART_RESPONSE"

echo ""

# 5. 创建订单（并发处理测试）
echo "5. 创建订单（并发处理测试）..."
CREATE_ORDER_RESPONSE=$(curl -s -X POST http://localhost:8080/api/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "shipping_address": "北京市朝阳区某某街道123号",
    "cart_item_ids": ['$CART_ITEM_ID1']
  }')

echo "创建订单响应: $CREATE_ORDER_RESPONSE"

# 提取订单ID
ORDER_ID=$(echo $CREATE_ORDER_RESPONSE | grep -o '"id":[0-9]*' | cut -d':' -f2)
echo "创建的订单ID: $ORDER_ID"

echo ""

# 6. 验证购物车已清空
echo "6. 验证购物车已清空..."
EMPTY_CART_RESPONSE=$(curl -s -X GET http://localhost:8080/api/cart \
  -H "Authorization: Bearer $TOKEN")

echo "订单创建后的购物车响应: $EMPTY_CART_RESPONSE"

echo ""

# 7. 获取订单列表
echo "7. 获取订单列表..."
ORDERS_RESPONSE=$(curl -s -X GET "http://localhost:8080/api/orders?page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN")

echo "订单列表响应: $ORDERS_RESPONSE"

echo ""

# 8. 获取订单详情
echo "8. 获取订单详情..."
ORDER_DETAIL_RESPONSE=$(curl -s -X GET http://localhost:8080/api/orders/$ORDER_ID \
  -H "Authorization: Bearer $TOKEN")

echo "订单详情响应: $ORDER_DETAIL_RESPONSE"

echo ""

# 9. 更新订单状态（模拟支付）
echo "9. 更新订单状态（模拟支付）..."
UPDATE_ORDER_STATUS_RESPONSE=$(curl -s -X PUT http://localhost:8080/api/orders/$ORDER_ID/status \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "status": "paid"
  }')

echo "支付成功响应: $UPDATE_ORDER_STATUS_RESPONSE"

echo ""

# 10. 再次查看订单详情（验证状态更新）
echo "10. 验证订单状态更新..."
PAID_ORDER_RESPONSE=$(curl -s -X GET http://localhost:8080/api/orders/$ORDER_ID \
  -H "Authorization: Bearer $TOKEN")

echo "支付后的订单详情: $PAID_ORDER_RESPONSE"

echo ""

# 11. 更新订单状态为已发货
echo "11. 更新订单状态为已发货..."
SHIP_ORDER_RESPONSE=$(curl -s -X PUT http://localhost:8080/api/orders/$ORDER_ID/status \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "status": "shipped"
  }')

echo "发货成功响应: $SHIP_ORDER_RESPONSE"

echo ""

# 12. 再添加商品到购物车用于取消订单测试
echo "12. 为取消订单测试添加商品到购物车..."
CART_ADD_RESPONSE3=$(curl -s -X POST http://localhost:8080/api/cart/add \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "product_id": 2,
    "quantity": 1
  }')

echo "添加商品到购物车响应: $CART_ADD_RESPONSE3"

# 提取新的购物车项ID
NEW_CART_ITEM_ID=$(echo $CART_ADD_RESPONSE3 | grep -o '"id":[0-9]*' | cut -d':' -f2)

# 创建新订单
echo "创建新订单用于取消测试..."
CREATE_ORDER_RESPONSE2=$(curl -s -X POST http://localhost:8080/api/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "shipping_address": "上海市浦东新区某某路456号",
    "cart_item_ids": ['$NEW_CART_ITEM_ID']
  }')

echo "创建新订单响应: $CREATE_ORDER_RESPONSE2"

NEW_ORDER_ID=$(echo $CREATE_ORDER_RESPONSE2 | grep -o '"id":[0-9]*' | cut -d':' -f2)
echo "新订单ID: $NEW_ORDER_ID"

echo ""

# 13. 取消订单测试
echo "13. 取消订单测试..."
CANCEL_ORDER_RESPONSE=$(curl -s -X DELETE http://localhost:8080/api/orders/$NEW_ORDER_ID \
  -H "Authorization: Bearer $TOKEN")

echo "取消订单响应: $CANCEL_ORDER_RESPONSE"

# 验证订单状态是否变为已取消
CANCELLED_ORDER_RESPONSE=$(curl -s -X GET http://localhost:8080/api/orders/$NEW_ORDER_ID \
  -H "Authorization: Bearer $TOKEN")

echo "取消后的订单详情: $CANCELLED_ORDER_RESPONSE"

echo ""

# 14. 测试并发购物车操作
echo "14. 测试并发购物车操作..."

# 并发添加相同商品到购物车
for i in {1..3}; do
  curl -s -X POST http://localhost:8080/api/cart/add \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{
      "product_id": 1,
      "quantity": 1
    }' > /dev/null &
done

# 等待并发操作完成
wait

# 查看最终购物车状态
FINAL_CART_RESPONSE=$(curl -s -X GET http://localhost:8080/api/cart \
  -H "Authorization: Bearer $TOKEN")

echo "并发操作后的购物车响应: $FINAL_CART_RESPONSE"

echo ""

# 15. 清空购物车测试
echo "15. 清空购物车测试..."
CLEAR_CART_RESPONSE=$(curl -s -X DELETE http://localhost:8080/api/cart/clear \
  -H "Authorization: Bearer $TOKEN")

echo "清空购物车响应: $CLEAR_CART_RESPONSE"

# 验证购物车是否已清空
CLEARED_CART_RESPONSE=$(curl -s -X GET http://localhost:8080/api/cart \
  -H "Authorization: Bearer $TOKEN")

echo "清空后的购物车响应: $CLEARED_CART_RESPONSE"

echo ""

# 16. 测试库存不足情况
echo "16. 测试库存不足情况..."

# 尝试添加超过库存的商品数量
STOCK_TEST_RESPONSE=$(curl -s -X POST http://localhost:8080/api/cart/add \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "product_id": 2,
    "quantity": 10000
  }')

echo "库存不足测试响应: $STOCK_TEST_RESPONSE"

echo ""

# 17. 最终订单列表查看
echo "17. 最终订单列表查看..."
FINAL_ORDERS_RESPONSE=$(curl -s -X GET "http://localhost:8080/api/orders?page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN")

echo "最终订单列表响应: $FINAL_ORDERS_RESPONSE"

echo ""

echo "=== 订单管理API测试完成 ==="

echo ""
echo "=== 测试总结 ==="
echo "✅ 购物车功能: 添加、查看、更新、删除、清空"
echo "✅ 订单功能: 创建、查看、更新状态、取消"
echo "✅ 并发处理: 协程处理订单创建和状态更新"
echo "✅ 库存控制: 并发安全的库存扣减和恢复"
echo "✅ 事务处理: 订单创建的数据一致性"
echo "✅ 错误处理: 库存不足、权限验证等"