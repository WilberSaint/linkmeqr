ALTER TABLE profile_blocks
    MODIFY block_type ENUM('instagram','facebook','tiktok','youtube','whatsapp','phone','email',
                            'location','website','menu','catalog','image','video','text','link')
                            NOT NULL;
