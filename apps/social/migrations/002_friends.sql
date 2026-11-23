CREATE TABLE IF NOT EXISTS `friendships` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_a` VARCHAR(64) NOT NULL,
    `user_b` VARCHAR(64) NOT NULL,
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '1 active, 2 deleted, 3 blocked',
    `version` BIGINT UNSIGNED NOT NULL DEFAULT 1,
    `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_friendship_pair` (`user_a`, `user_b`),
    KEY `idx_friendship_b_status` (`user_b`, `status`),
    KEY `idx_friendship_status_updated` (`status`, `updated_at`),
    CONSTRAINT `chk_friendship_order` CHECK (`user_a` < `user_b`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `friend_requests` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `requester_id` VARCHAR(64) NOT NULL,
    `recipient_id` VARCHAR(64) NOT NULL,
    `message` VARCHAR(255) NOT NULL DEFAULT '',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '1 pending, 2 accepted, 3 rejected, 4 canceled',
    `pending_pair` VARCHAR(130) GENERATED ALWAYS AS (
        CASE
            WHEN `status` = 1 THEN CONCAT(
                LEAST(`requester_id`, `recipient_id`),
                ':',
                GREATEST(`requester_id`, `recipient_id`)
            )
            ELSE NULL
        END
    ) STORED,
    `handled_at` DATETIME(3) NULL,
    `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_friend_request_pending` (`pending_pair`),
    KEY `idx_friend_request_recipient` (`recipient_id`, `status`, `created_at`),
    KEY `idx_friend_request_requester` (`requester_id`, `status`, `created_at`),
    CONSTRAINT `chk_friend_request_users` CHECK (`requester_id` <> `recipient_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;