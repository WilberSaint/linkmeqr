-- LinkMeQR Studio now doubles as a lightweight sales tracker for the
-- physical cards admin designs and sells per client: each card moves
-- through draft -> printed -> delivered, with an optional free-text note
-- (e.g. what was charged, who picked it up).

ALTER TABLE print_cards
    ADD COLUMN status    VARCHAR(20)  NOT NULL DEFAULT 'draft' AFTER content,
    ADD COLUMN sale_note VARCHAR(500) NULL     AFTER status;
