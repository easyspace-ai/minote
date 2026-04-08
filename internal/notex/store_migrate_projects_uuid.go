package notex

import (
	"context"
	"fmt"
)

// recreateProjectsAndMaterialsUUIDSchema drops and recreates project tables with uuid PK.
// Used when upgrading from legacy bigint project ids (dev wipe).
const recreateProjectsAndMaterialsUUIDSchema = `
create table notex_projects (
	id uuid primary key,
	user_id bigint not null references notex_users(id) on delete cascade,
	library_id bigint not null references notex_libraries(id),
	name text not null,
	description text not null default '',
	category text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	starred boolean not null default false,
	archived boolean not null default false,
	icon_index integer not null default -1,
	accent_hex text not null default '',
	studio_scope jsonb not null default '{}'::jsonb
);
create index idx_notex_projects_user_id on notex_projects(user_id);
create index idx_notex_projects_user_id_updated on notex_projects(user_id, updated_at DESC);

create table notex_materials (
	id bigserial primary key,
	project_id uuid not null references notex_projects(id) on delete cascade,
	kind text not null,
	title text not null,
	status text not null default 'pending',
	subtitle text not null default '',
	payload jsonb not null default '{}'::jsonb,
	file_path text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);
create index idx_notex_materials_project_id on notex_materials(project_id);
create index idx_notex_materials_project_id_created on notex_materials(project_id, created_at DESC);
`

func (s *Store) migrateLegacyProjectBigintSchema(ctx context.Context) error {
	if s == nil || s.db == nil {
		return nil
	}
	var dt string
	err := s.db.QueryRow(ctx, `
		select data_type::text
		from information_schema.columns
		where table_schema = 'public' and table_name = 'notex_projects' and column_name = 'id'
	`).Scan(&dt)
	if err != nil {
		if isNoRows(err) {
			return nil
		}
		return fmt.Errorf("inspect notex_projects.id: %w", err)
	}
	if dt == "uuid" {
		return nil
	}
	if dt != "bigint" && dt != "integer" {
		return fmt.Errorf("notex_projects.id has unexpected type %q; drop notex_materials and notex_projects manually to continue", dt)
	}
	if _, err := s.db.Exec(ctx, `drop table if exists notex_materials cascade`); err != nil {
		return fmt.Errorf("drop legacy notex_materials: %w", err)
	}
	if _, err := s.db.Exec(ctx, `drop table if exists notex_projects cascade`); err != nil {
		return fmt.Errorf("drop legacy notex_projects: %w", err)
	}
	if _, err := s.db.Exec(ctx, recreateProjectsAndMaterialsUUIDSchema); err != nil {
		return fmt.Errorf("recreate notex project tables with uuid: %w", err)
	}
	return nil
}
