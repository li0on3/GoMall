-- GoMall 数据库初始化脚本

-- 创建数据库
CREATE DATABASE IF NOT EXISTS gomall CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 使用数据库
USE gomall;

-- 创建用户（如果不存在）
CREATE USER IF NOT EXISTS 'gomall_user'@'localhost' IDENTIFIED BY 'gomall_password_2024';

-- 授权
GRANT ALL PRIVILEGES ON gomall.* TO 'gomall_user'@'localhost';
GRANT ALL PRIVILEGES ON gomall.* TO 'test'@'localhost';

-- 刷新权限
FLUSH PRIVILEGES;

-- 显示创建结果
SHOW DATABASES LIKE 'gomall';
SELECT User, Host FROM mysql.user WHERE User IN ('gomall_user', 'test');