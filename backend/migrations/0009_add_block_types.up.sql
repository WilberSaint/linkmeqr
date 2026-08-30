-- Google review CTA, plus richer content-driven blocks (gallery, hours,
-- testimonials, embedded map) whose data lives in the already-existing but
-- previously-unused profile_blocks.content JSON column.
ALTER TABLE profile_blocks
    MODIFY block_type ENUM('instagram','facebook','tiktok','youtube','whatsapp','phone','email',
                            'location','website','menu','catalog','image','video','text','link',
                            'google_review','gallery','hours','testimonials','map')
                            NOT NULL;
