package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luqmanulhakimdev/tykket-go/internal/domain"
)

type rbacRepository struct {
	pool *pgxpool.Pool
}

func NewRBACRepository(pool *pgxpool.Pool) domain.RBACRepository {
	return &rbacRepository{pool: pool}
}

func (r *rbacRepository) GetRoleByName(ctx context.Context, name string) (*domain.Role, error) {
	query := `SELECT id, name, description, created_at FROM roles WHERE UPPER(name) = UPPER($1)`
	var role domain.Role
	err := r.pool.QueryRow(ctx, query, name).Scan(
		&role.ID,
		&role.Name,
		&role.Description,
		&role.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get role by name: %w", err)
	}
	return &role, nil
}

func (r *rbacRepository) GetRoleByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	query := `SELECT id, name, description, created_at FROM roles WHERE id = $1`
	var role domain.Role
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&role.ID,
		&role.Name,
		&role.Description,
		&role.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get role by id: %w", err)
	}
	return &role, nil
}

func (r *rbacRepository) ListRoles(ctx context.Context) ([]domain.Role, error) {
	query := `SELECT id, name, description, created_at FROM roles ORDER BY name ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	var roles []domain.Role
	for rows.Next() {
		var role domain.Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (r *rbacRepository) HasPermission(ctx context.Context, userID, projectID uuid.UUID, permissionCode string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM projects WHERE id = $2 AND owner_id = $1 AND deleted_at IS NULL
			UNION
			SELECT 1
			FROM project_members pm
			JOIN role_permissions rp ON pm.role_id = rp.role_id
			JOIN permissions p ON rp.permission_id = p.id
			JOIN projects pr ON pm.project_id = pr.id AND pr.deleted_at IS NULL
			WHERE pm.project_id = $2 AND pm.user_id = $1 AND pm.deleted_at IS NULL AND p.code = $3
		)
	`
	var hasPerm bool
	err := r.pool.QueryRow(ctx, query, userID, projectID, permissionCode).Scan(&hasPerm)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}
	return hasPerm, nil
}

func (r *rbacRepository) GetUserMemberInProject(ctx context.Context, userID, projectID uuid.UUID) (*domain.ProjectMember, error) {
	query := `
		SELECT 
			pm.id, pm.project_id, pm.user_id, pm.role_id, pm.joined_at, pm.deleted_at,
			u.id, u.email, u.name, u.avatar_url,
			r.id, r.name, r.description, r.created_at
		FROM project_members pm
		JOIN users u ON pm.user_id = u.id AND u.deleted_at IS NULL
		JOIN roles r ON pm.role_id = r.id
		WHERE pm.project_id = $1 AND pm.user_id = $2 AND pm.deleted_at IS NULL
	`
	var (
		pm domain.ProjectMember
		u  domain.User
		rl domain.Role
	)
	err := r.pool.QueryRow(ctx, query, projectID, userID).Scan(
		&pm.ID,
		&pm.ProjectID,
		&pm.UserID,
		&pm.RoleID,
		&pm.JoinedAt,
		&pm.DeletedAt,
		&u.ID,
		&u.Email,
		&u.Name,
		&u.AvatarURL,
		&rl.ID,
		&rl.Name,
		&rl.Description,
		&rl.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get project member: %w", err)
	}

	pm.User = &u
	pm.Role = &rl
	return &pm, nil
}

func (r *rbacRepository) ListProjectMembers(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectMember, error) {
	query := `
		SELECT 
			pm.id, pm.project_id, pm.user_id, pm.role_id, pm.joined_at, pm.deleted_at,
			u.id, u.email, u.name, u.avatar_url,
			r.id, r.name, r.description, r.created_at
		FROM project_members pm
		JOIN users u ON pm.user_id = u.id AND u.deleted_at IS NULL
		JOIN roles r ON pm.role_id = r.id
		WHERE pm.project_id = $1 AND pm.deleted_at IS NULL
		ORDER BY pm.joined_at ASC
	`
	rows, err := r.pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query project members: %w", err)
	}
	defer rows.Close()

	var members []domain.ProjectMember
	for rows.Next() {
		var (
			pm domain.ProjectMember
			u  domain.User
			rl domain.Role
		)
		if err := rows.Scan(
			&pm.ID,
			&pm.ProjectID,
			&pm.UserID,
			&pm.RoleID,
			&pm.JoinedAt,
			&pm.DeletedAt,
			&u.ID,
			&u.Email,
			&u.Name,
			&u.AvatarURL,
			&rl.ID,
			&rl.Name,
			&rl.Description,
			&rl.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan project member: %w", err)
		}
		pm.User = &u
		pm.Role = &rl
		members = append(members, pm)
	}
	return members, nil
}

func (r *rbacRepository) AddProjectMember(ctx context.Context, m *domain.ProjectMember) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}

	query := `
		INSERT INTO project_members (id, project_id, user_id, role_id, joined_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING joined_at
	`
	err := r.pool.QueryRow(ctx, query, m.ID, m.ProjectID, m.UserID, m.RoleID).Scan(&m.JoinedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrConflict
		}
		return fmt.Errorf("failed to add project member: %w", err)
	}
	return nil
}

func (r *rbacRepository) UpdateProjectMemberRole(ctx context.Context, projectID, userID, roleID uuid.UUID) error {
	query := `
		UPDATE project_members
		SET role_id = $3
		WHERE project_id = $1 AND user_id = $2 AND deleted_at IS NULL
	`
	ct, err := r.pool.Exec(ctx, query, projectID, userID, roleID)
	if err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *rbacRepository) RemoveProjectMember(ctx context.Context, projectID, userID uuid.UUID) error {
	query := `UPDATE project_members SET deleted_at = NOW() WHERE project_id = $1 AND user_id = $2 AND deleted_at IS NULL`
	ct, err := r.pool.Exec(ctx, query, projectID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

