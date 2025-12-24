-- init.sql

-- 创建 users 表
CREATE TABLE IF NOT EXISTS `users` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `created_at` DATETIME NOT NULL,
    `updated_at` DATETIME NOT NULL,
    `deleted_at` DATETIME NULL,
    `name` VARCHAR(255) NOT NULL UNIQUE,
    `email` VARCHAR(100) UNIQUE,
    `age` INT DEFAULT 18,
    `password` VARCHAR(255) DEFAULT '123456',
    `is_verified` TINYINT(1) DEFAULT 0,
    PRIMARY KEY (`id`),
    INDEX idx_users_deleted_at (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 创建 market_prices 表
CREATE TABLE IF NOT EXISTS `market_prices` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `pro_id` BIGINT UNSIGNED NOT NULL,
    `pro_name` VARCHAR(255) NOT NULL,
    `market_id` BIGINT UNSIGNED NOT NULL,
    `market_name` VARCHAR(255) NOT NULL,
    `price` DECIMAL(10,2) NOT NULL,
    `price_unit` VARCHAR(50) NOT NULL,
    `specifici_val` TEXT,
    `price_date` DATE NOT NULL,
    `create_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    INDEX idx_market_prices_pro_id (`pro_id`),
    INDEX idx_market_prices_market_id (`market_id`),
    INDEX idx_market_prices_price_date (`price_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;