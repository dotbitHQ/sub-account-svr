CREATE DATABASE `sub_account_db`;
USE `sub_account_db`;

-- t_block_parser_info
CREATE TABLE `t_block_parser_info`
(
    `id`           BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '',
    `parser_type`  SMALLINT            NOT NULL DEFAULT '0' COMMENT 'register-99 sub-acc-98 ckb-0 eth-1 btc-2 tron-3 bsc-5 4-wx polygon-6',
    `block_number` BIGINT(20) UNSIGNED NOT NULL DEFAULT '0' COMMENT '',
    `block_hash`   VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `parent_hash`  VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `created_at`   TIMESTAMP           NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '',
    `updated_at`   TIMESTAMP           NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '',
    PRIMARY KEY (`id`) USING BTREE,
    UNIQUE KEY `uk_parser_number` (parser_type, block_number) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT ='for block parser';

-- t_task_info
CREATE TABLE `t_task_info`
(
    `id`                BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '',
    `task_id`           VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `task_type`         SMALLINT            NOT NULL DEFAULT '0' COMMENT '0-delegate 1-normal 2-chain 3-closed',
    `parent_account_id` VARCHAR(255)        NOT NULL DEFAULT '' COMMENT 'smt tree',
    `action`            VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `ref_outpoint`      VARCHAR(255)        NOT NULL DEFAULT '' COMMENT 'ref sub account cell outpoint',
    `block_number`      BIGINT(20) UNSIGNED NOT NULL DEFAULT '0' COMMENT 'tx block number',
    `outpoint`          VARCHAR(255)        NOT NULL DEFAULT '' COMMENT 'new sub account cell outpoint',
    `timestamp`         BIGINT              NOT NULL DEFAULT '0' COMMENT 'record timestamp',
    `smt_status`        SMALLINT            NOT NULL DEFAULT '0' COMMENT 'smt status',
    `tx_status`         SMALLINT            NOT NULL DEFAULT '0' COMMENT 'tx status',
    `retry`             SMALLINT            NOT NULL DEFAULT '0' COMMENT '',
    `created_at`        TIMESTAMP           NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '',
    `updated_at`        TIMESTAMP           NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_task_id` (`task_id`),
    KEY `k_parent_account_id` (`parent_account_id`),
    KEY `k_ref_outpoint` (`ref_outpoint`),
    KEY `k_outpoint` (`outpoint`),
    KEY `k_smt_tx` (`smt_status`, `tx_status`),
    KEY `k_task_type` (`task_type`),
    KEY `k_action` (`action`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT ='task info';

-- t_smt_record_info
CREATE TABLE `t_smt_record_info`
(
    `id`                BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '',
    `account_id`        VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `nonce`             INT                 NOT NULL DEFAULT '0' COMMENT '',
    `record_type`       SMALLINT            NOT NULL DEFAULT '0' COMMENT '0-normal 1-closed 2-chain',
    `task_id`           VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `action`            VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `parent_account_id` VARCHAR(255)        NOT NULL DEFAULT '' COMMENT 'smt tree',
    `account`           VARCHAR(255)        NOT NULL DEFAULT '0' COMMENT '',
    `register_years`    INT                 NOT NULL DEFAULT '0' COMMENT '',
    `register_args`     VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `edit_key`          VARCHAR(255)        NOT NULL DEFAULT '' COMMENT 'owner,manager,records',
    `signature`         VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `edit_args`         VARCHAR(255)        NOT NULL DEFAULT '' COMMENT '',
    `edit_records`      TEXT                NOT NULL COMMENT '',
    `renew_years`       INT                 NOT NULL DEFAULT '0' COMMENT '',
    `timestamp`         BIGINT              NOT NULL DEFAULT '0' COMMENT 'record timestamp',
    `created_at`        TIMESTAMP           NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '',
    `updated_at`        TIMESTAMP           NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_account_nonce` (`account_id`, `nonce`, `record_type`),
    KEY `k_task_id` (`task_id`),
    KEY `k_parent_account_id` (`parent_account_id`),
    KEY `k_account` (`account`),
    KEY `k_action` (`action`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT ='smt record info';
