package dbhelper

import (
	db "khiladiBuzz/database"
	model "khiladiBuzz/models"
)

func GetPlayerByUserID(userID string) (model.Player, error) {
	var player model.Player
	query := `
		SELECT id, user_id, player_name, phone_number, role, batting_style, bowling_style, created_at, updated_at
		FROM players
		WHERE user_id = $1
	`
	err := db.KhiladiDb.Get(&player, query, userID)
	return player, err
}

func UpdatePlayerProfile(userID string, name, phone, role string, batting, bowling *string) error {
	query := `
		UPDATE players
		SET player_name = $1, phone_number = $2, role = $3::player_role_enum, batting_style = $4::batting_style_enum, bowling_style = $5::bowling_style_enum, updated_at = NOW()
		WHERE user_id = $6
	`
	_, err := db.KhiladiDb.Exec(query, name, phone, role, batting, bowling, userID)
	if err != nil {
		return err
	}
	
	// Also sync user name in users table!
	userQuery := `
		UPDATE users
		SET name = $1, phone_number = $2, updated_at = NOW()
		WHERE id = $3
	`
	_, err = db.KhiladiDb.Exec(userQuery, name, phone, userID)
	return err
}
