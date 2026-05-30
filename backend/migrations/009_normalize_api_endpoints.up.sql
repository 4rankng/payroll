-- Create api_endpoints table to store unique endpoint definitions
CREATE TABLE `api_endpoints` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `method` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `path` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_method_path` (`method`, `path`),
  KEY `idx_method_path` (`method`, `path`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Populate api_endpoints with unique method/path combinations from existing api_metrics
INSERT INTO `api_endpoints` (`method`, `path`)
SELECT DISTINCT `method`, `path`
FROM `api_metrics`
ORDER BY `method`, `path`;

-- Add endpoint_id column to api_metrics (nullable initially for data migration)
ALTER TABLE `api_metrics`
ADD COLUMN `endpoint_id` bigint unsigned NULL AFTER `id`;

-- Populate endpoint_id by matching with api_endpoints
UPDATE `api_metrics` am
INNER JOIN `api_endpoints` ae ON am.method = ae.method AND am.path = ae.path
SET am.endpoint_id = ae.id;

-- Make endpoint_id NOT NULL now that all records are populated
ALTER TABLE `api_metrics`
MODIFY COLUMN `endpoint_id` bigint unsigned NOT NULL;

-- Drop the old method and path columns (now redundant)
ALTER TABLE `api_metrics`
DROP INDEX `idx_method_path`,
DROP COLUMN `method`,
DROP COLUMN `path`;

-- Add foreign key constraint
ALTER TABLE `api_metrics`
ADD CONSTRAINT `fk_api_metrics_endpoint`
FOREIGN KEY (`endpoint_id`) REFERENCES `api_endpoints` (`id`)
ON DELETE RESTRICT ON UPDATE CASCADE;

-- Add index on endpoint_id for performance
ALTER TABLE `api_metrics`
ADD INDEX `idx_endpoint_id` (`endpoint_id`);
