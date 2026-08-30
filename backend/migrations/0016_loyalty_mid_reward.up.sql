-- Optional intermediate reward tier, alongside the existing final reward at
-- stamps_required. Both thresholds are business-editable. mid_reward_stamps
-- being non-null is the "intermediate reward enabled" flag — reaching it is
-- a purely derived, read-time milestone (stamps_count vs mid_reward_stamps),
-- never a stored per-customer flag, so it never blocks stamping or resets
-- the card. Only redeeming the final reward resets stamps_count.

ALTER TABLE loyalty_programs
    ADD COLUMN mid_reward_stamps      INT          NULL AFTER stamps_required,
    ADD COLUMN mid_reward_description VARCHAR(255) NULL AFTER mid_reward_stamps;
