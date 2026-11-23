CREATE TABLE IF NOT EXISTS `chat_groups` (
    `id` VARCHAR(64) NOT NULL,
    `name` VARCHAR(128) NOT NULL,
    `avatar_url` VARCHAR(512) NOT NULL DEFAULT '',
    `notification` VARCHAR(512) NOT NULL DEFAULT '',
    `owner_id` VARCHAR(64) NOT NULL,
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '1 active, 2 disabled, 3 dissolved',
    `membership_version` BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (`id`),
    KEY `idx_groups_owner_status` (`owner_id`, `status`),
    KEY `idx_groups_status_updated` (`status`, `updated_at`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `group_members` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `group_id` VARCHAR(64) NOT NULL,
    `user_id` VARCHAR(64) NOT NULL,
    `role` TINYINT NOT NULL DEFAULT 3 COMMENT '1 owner, 2 admin, 3 member',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '1 active, 2 left, 3 kicked',
    `joined_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    `left_at` DATETIME(3) NULL,
    `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_group_member` (`group_id`, `user_id`),
    KEY `idx_member_user_status` (`user_id`, `status`),
    KEY `idx_member_group_status_role` (`group_id`, `status`, `role`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `group_members` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `group_id` VARCHAR(64) NOT NULL,
    `user_id` VARCHAR(64) NOT NULL,
    `role` TINYINT NOT NULL DEFAULT 3 COMMENT '1 owner, 2 admin, 3 member',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '1 active, 2 left, 3 kicked',
    `joined_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    `left_at` DATETIME(3) NULL,
    `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_group_member` (`group_id`, `user_id`),
    KEY `idx_member_user_status` (`user_id`, `status`),
    KEY `idx_member_group_status_role` (`group_id`, `status`, `role`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `social_outbox` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `event_id` VARCHAR(64) NOT NULL,
    `aggregate_type` VARCHAR(32) NOT NULL,
    `aggregate_id` VARCHAR(64) NOT NULL,
    `event_type` VARCHAR(64) NOT NULL,
    `event_version` BIGINT UNSIGNED NOT NULL,
    `payload` JSON NOT NULL,
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '1 pending, 2 processing, 3 sent, 4 failed',
    `attempts` INT UNSIGNED NOT NULL DEFAULT 0,
    `available_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    `locked_until` DATETIME(3) NULL,
    `sent_at` DATETIME(3) NULL,
    `last_error` VARCHAR(1024) NOT NULL DEFAULT '',
    `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_social_outbox_event` (`event_id`),
    UNIQUE KEY `uk_social_outbox_version` (
        `aggregate_type`,
        `aggregate_id`,
        `event_version`
    ),
    KEY `idx_social_outbox_pending` (`status`, `available_at`),
    KEY `idx_social_outbox_lock` (`status`, `locked_until`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;