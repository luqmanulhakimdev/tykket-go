DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'project_type_enum') THEN
        CREATE TYPE project_type_enum AS ENUM ('SCRUM', 'KANBAN');
    END IF;
END $$;

ALTER TABLE projects ADD COLUMN IF NOT EXISTS type project_type_enum NOT NULL DEFAULT 'SCRUM';

