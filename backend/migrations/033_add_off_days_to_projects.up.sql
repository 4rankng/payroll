  ALTER TABLE projects
      ADD COLUMN off_days TINYINT UNSIGNED NOT NULL DEFAULT 0
          COMMENT 'Bitmask of weekly off days: bit0=Sun,bit1=Mon,...,bit6=Sat. Default 0 = no fixed off days';