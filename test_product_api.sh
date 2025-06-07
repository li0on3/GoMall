#!/bin/bash

# 测试商品管理API脚本

echo "=== GoMall 商品管理API测试 ==="

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

# 1. 创建商品分类
echo "1. 创建商品分类..."
CATEGORY_RESPONSE=$(curl -s -X POST http://localhost:8080/api/categories \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "数码产品",
    "description": "各种数码电子产品",
    "parent_id": 0,
    "sort_order": 1
  }')

echo "分类创建响应: $CATEGORY_RESPONSE"

# 提取分类ID
CATEGORY_ID=$(echo $CATEGORY_RESPONSE | grep -o '"id":[0-9]*' | cut -d':' -f2 | head -1)
echo "创建的分类ID: $CATEGORY_ID"

echo ""

# 2. 获取分类列表
echo "2. 获取分类列表..."
CATEGORIES_RESPONSE=$(curl -s -X GET http://localhost:8080/api/categories)
echo "分类列表响应: $CATEGORIES_RESPONSE"

echo ""

# 3. 创建子分类
echo "3. 创建子分类..."
SUBCATEGORY_RESPONSE=$(curl -s -X POST http://localhost:8080/api/categories \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "智能手机",
    "description": "各品牌智能手机",
    "parent_id": '$CATEGORY_ID',
    "sort_order": 1
  }')

echo "子分类创建响应: $SUBCATEGORY_RESPONSE"

SUBCATEGORY_ID=$(echo $SUBCATEGORY_RESPONSE | grep -o '"id":[0-9]*' | cut -d':' -f2 | head -1)
echo "创建的子分类ID: $SUBCATEGORY_ID"

echo ""

# 4. 创建商品
echo "4. 创建商品..."
PRODUCT_RESPONSE=$(curl -s -X POST http://localhost:8080/api/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "iPhone 15 Pro",
    "description": "苹果最新款智能手机，搭载A17芯片",
    "price": 7999.00,
    "stock": 100,
    "category_id": '$SUBCATEGORY_ID',
    "images": ["/upload/products/iphone15pro_1.jpg", "/upload/products/iphone15pro_2.jpg"]
  }')

echo "商品创建响应: $PRODUCT_RESPONSE"

PRODUCT_ID=$(echo $PRODUCT_RESPONSE | grep -o '"id":[0-9]*' | cut -d':' -f2 | head -1)
echo "创建的商品ID: $PRODUCT_ID"

echo ""

# 5. 再创建几个商品用于测试
echo "5. 创建更多测试商品..."

# 创建第二个商品
curl -s -X POST http://localhost:8080/api/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Samsung Galaxy S24",
    "description": "三星旗舰智能手机",
    "price": 6999.00,
    "stock": 80,
    "category_id": '$SUBCATEGORY_ID',
    "images": ["/upload/products/galaxy_s24.jpg"]
  }' > /dev/null

# 创建第三个商品
curl -s -X POST http://localhost:8080/api/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "小米14 Pro",
    "description": "小米高端智能手机",
    "price": 4999.00,
    "stock": 120,
    "category_id": '$SUBCATEGORY_ID',
    "images": []
  }' > /dev/null

echo "已创建3个测试商品"

echo ""

# 6. 获取商品列表
echo "6. 获取商品列表..."
PRODUCTS_RESPONSE=$(curl -s -X GET "http://localhost:8080/api/products?page=1&page_size=10")
echo "商品列表响应: $PRODUCTS_RESPONSE"

echo ""

# 7. 获取商品详情
echo "7. 获取商品详情..."
PRODUCT_DETAIL_RESPONSE=$(curl -s -X GET http://localhost:8080/api/products/$PRODUCT_ID)
echo "商品详情响应: $PRODUCT_DETAIL_RESPONSE"

echo ""

# 8. 搜索商品
echo "8. 搜索商品（关键字：iPhone）..."
SEARCH_RESPONSE=$(curl -s -X GET "http://localhost:8080/api/products/search?keyword=iPhone&page=1&page_size=5")
echo "搜索响应: $SEARCH_RESPONSE"

echo ""

# 9. 按分类筛选商品
echo "9. 按分类筛选商品..."
FILTER_RESPONSE=$(curl -s -X GET "http://localhost:8080/api/products?category_id=$SUBCATEGORY_ID&page=1&page_size=10")
echo "分类筛选响应: $FILTER_RESPONSE"

echo ""

# 10. 价格范围筛选
echo "10. 价格范围筛选（5000-8000）..."
PRICE_FILTER_RESPONSE=$(curl -s -X GET "http://localhost:8080/api/products?min_price=5000&max_price=8000&page=1&page_size=10")
echo "价格筛选响应: $PRICE_FILTER_RESPONSE"

echo ""

# 11. 获取热门商品
echo "11. 获取热门商品..."
HOT_PRODUCTS_RESPONSE=$(curl -s -X GET "http://localhost:8080/api/products/hot?limit=5")
echo "热门商品响应: $HOT_PRODUCTS_RESPONSE"

echo ""

# 12. 更新商品
echo "12. 更新商品信息..."
UPDATE_RESPONSE=$(curl -s -X PUT http://localhost:8080/api/products/$PRODUCT_ID \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "iPhone 15 Pro Max",
    "price": 8999.00,
    "stock": 90
  }')

echo "商品更新响应: $UPDATE_RESPONSE"

echo ""

# 13. 更新分类
echo "13. 更新分类信息..."
UPDATE_CATEGORY_RESPONSE=$(curl -s -X PUT http://localhost:8080/api/categories/$SUBCATEGORY_ID \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "高端智能手机",
    "description": "各品牌高端智能手机产品"
  }')

echo "分类更新响应: $UPDATE_CATEGORY_RESPONSE"

echo ""

# 14. 测试商品排序
echo "14. 测试商品排序（按价格升序）..."
SORT_RESPONSE=$(curl -s -X GET "http://localhost:8080/api/products?sort_by=price&sort_order=asc&page=1&page_size=10")
echo "排序响应: $SORT_RESPONSE"

echo ""

# 15. 创建测试图片文件并上传
echo "15. 测试图片上传..."

# 创建一个测试图片文件
echo "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==" | base64 -d > /tmp/test_image.png

UPLOAD_RESPONSE=$(curl -s -X POST http://localhost:8080/api/upload/images \
  -H "Authorization: Bearer $TOKEN" \
  -F "images=@/tmp/test_image.png")

echo "图片上传响应: $UPLOAD_RESPONSE"

# 清理测试文件
rm -f /tmp/test_image.png

echo ""

# 16. 测试删除商品（软删除）
echo "16. 测试删除商品..."
DELETE_RESPONSE=$(curl -s -X DELETE http://localhost:8080/api/products/$PRODUCT_ID \
  -H "Authorization: Bearer $TOKEN")

echo "商品删除响应: $DELETE_RESPONSE"

# 验证商品是否被软删除
DELETED_CHECK=$(curl -s -X GET http://localhost:8080/api/products/$PRODUCT_ID)
echo "删除后查询响应: $DELETED_CHECK"

echo ""

echo "=== 商品管理API测试完成 ==="