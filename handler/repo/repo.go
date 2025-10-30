package repo

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"greateDateBot/model"
	"log"
)

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{
		db: db,
	}
}

func (r *Repo) CreateUser(ctx context.Context, username string, id int64) error {
	_, err := r.db.Exec(ctx, "INSERT INTO users (id, username, count) VALUES ($1, $2, 0)", id, username)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) GetUserCount(ctx context.Context, userID int64) (int, error) {
	row := r.db.QueryRow(ctx, "SELECT count FROM users WHERE id = $1", userID)
	var count int
	err := row.Scan(&count)
	if errors.Is(err, pgx.ErrNoRows) {
		log.Printf("user %d not found", userID)
		return -1, err
	}

	return count, nil
}

func (r *Repo) UpdateUserCount(ctx context.Context, userID int64) error {
	_, err := r.db.Exec(ctx, "UPDATE users SET count = count + 1 WHERE id = $1", userID)

	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) GetUser(ctx context.Context, userID int64) (*model.User, error) {
	user := model.User{}

	row := r.db.QueryRow(ctx, "SELECT id, username, count FROM users WHERE id = $1", userID)
	err := row.Scan(&user.ID, &user.Username, &user.Count)
	if errors.Is(err, pgx.ErrNoRows) {
		log.Printf("user %d not found", userID)
		return &model.User{}, err
	}

	return &user, err
}

func (r *Repo) GetUserByName(ctx context.Context, username string) (*model.User, error) {
	user := model.User{Username: username}

	row := r.db.QueryRow(ctx, "SELECT id, count FROM users WHERE username = $1", username)
	err := row.Scan(&user.ID, &user.Count)
	if errors.Is(err, pgx.ErrNoRows) {
		log.Printf("user %s not found", username)
		return &model.User{}, err
	}

	return &user, err
}

func (r *Repo) UserExists(ctx context.Context, userID int64) bool {
	row := r.db.QueryRow(ctx, "SELECT count(*) FROM users WHERE id = $1", userID)
	var count int
	err := row.Scan(&count)
	if errors.Is(err, pgx.ErrNoRows) {
		log.Printf("user %d not found", userID)
		return false
	}

	return count > 0
}

func (r *Repo) DeleteUser(ctx context.Context, userID int64) error {
	_, err := r.db.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repo) GetAllUsers(ctx context.Context) ([]model.User, error) {
	var users []model.User
	rows, err := r.db.Query(ctx, "SELECT id, username, count FROM users")
	if err != nil {
		log.Printf("GetAllUsers query error: %v", err)
		return users, err
	}

	log.Printf("GetAllUsers: query executed successfully")

	count := 0
	for rows.Next() {
		userRow := model.User{}
		err = rows.Scan(&userRow.ID, &userRow.Username, &userRow.Count)
		if err != nil {
			log.Printf("GetAllUsers scan error: %v", err)
			return users, err
		}
		users = append(users, userRow)
		count++
		log.Printf("GetAllUsers: added user %d: %s (count: %d)", userRow.ID, userRow.Username, userRow.Count)
	}

	log.Printf("GetAllUsers: total users found: %d", count)
	return users, nil
}
