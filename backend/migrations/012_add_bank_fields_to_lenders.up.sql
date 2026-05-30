-- Add bank fields to lenders table
ALTER TABLE `lenders` 
ADD COLUMN `bank_id` bigint unsigned NULL,
ADD COLUMN `bank_account_number` varchar(30) NULL,
ADD COLUMN `bank_account_name` varchar(255) NULL,
ADD KEY `fk_lenders_bank` (`bank_id`),
ADD CONSTRAINT `fk_lenders_bank` FOREIGN KEY (`bank_id`) REFERENCES `banks` (`id`);