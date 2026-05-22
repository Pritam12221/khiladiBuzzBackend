package dbhelper

import (
	db "khiladiBuzz/database"
	model "khiladiBuzz/models"
)

func GetPlayerByUserID(userID string) (model.Player, error) {
	var player model.Player
	query := `
		SELECT p.id, u.name AS player_name, u.phone_number, p.user_id, p.role, p.batting_style, p.bowling_style, p.created_at, p.updated_at,
		       p.career_matches, p.career_runs, p.career_balls_faced, p.career_fours, p.career_sixes,
		       p.career_wickets, p.career_balls_bowled, p.career_runs_given, p.career_wins
		FROM player_stats p
		JOIN users u ON p.user_id = u.id
		WHERE p.user_id = $1
	`
	err := db.KhiladiDb.Get(&player, query, userID)
	return player, err
}

func UpdatePlayerProfile(userID string, name, phone, role string, batting, bowling *string) error {

	tx, err := db.KhiladiDb.Beginx()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	// Update player_stats
	query := `
		UPDATE player_stats
		SET role = $1::player_role_enum, batting_style = $2::batting_style_enum, bowling_style = $3::bowling_style_enum, updated_at = NOW()
		WHERE user_id = $4
	`
	_, err = tx.Exec(query, role, batting, bowling, userID)
	if err != nil {
		return err
	}

	//sync user name and phone in users table
	userQuery := `
		UPDATE users
		SET name = $1, phone_number = $2, updated_at = NOW()
		WHERE id = $3
	`
	_, err = tx.Exec(userQuery, name, phone, userID)
	if err != nil {
		return err
	}
	return tx.Commit()
}
