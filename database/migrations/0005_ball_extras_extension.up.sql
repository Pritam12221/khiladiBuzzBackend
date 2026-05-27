BEGIN;

ALTER TYPE extra_type_enum ADD VALUE IF NOT EXISTS 'bye';
ALTER TYPE extra_type_enum ADD VALUE IF NOT EXISTS 'leg_bye';

COMMIT;
