package dbhelper

import (
	"database/sql"
	"errors"
	db "khiladiBuzz/database"
	model "khiladiBuzz/models"
	"khiladiBuzz/utils"
)

func GetUserByPhoneNumber(phoneNumber, password string) (model.User, error) {
	var user model.User
	query := `SELECT id, name, phone_number,password FROM users WHERE phone_number = $1 `

	err := db.KhiladiDb.Get(&user, query, phoneNumber)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, errors.New("no user exist")
		}
		return model.User{}, err
	}

	if err := utils.CheckPassword(user.Password, password); err != nil {
		return model.User{}, errors.New("invalid credentials")
	}

	return user, nil
}

func IsUserExist(phoneNumber string) (bool, error) {
	var exists bool
	query := `SELECT COUNT(*) > 0 FROM users WHERE phone_number = $1`
	err := db.KhiladiDb.Get(&exists, query, phoneNumber)
	return exists, err
}


func CreateUser(name, phoneNumber, password string) (string, error) {
	query := `INSERT INTO users(name, phone_number, password)
	VALUES ($1, TRIM(LOWER($2)), $3) RETURNING id;`

	var userID string
	err := db.KhiladiDb.Get(&userID, query, name, phoneNumber, password)
	return userID, err
}

func CreateUserSession(userID string) (string, error) {
	query := `INSERT INTO user_session(user_id)
			VALUES ($1) RETURNING id;`
	var sessionID string
	err := db.KhiladiDb.Get(&sessionID, query, userID)
	if err != nil {
		return "", err
	}
	return sessionID, nil
}

func GetUserIDBySession(sessionID string) (string, error) {
	var userID string

	query := `
		SELECT user_id 
		FROM user_session 
		WHERE id = $1 AND archived_at IS NULL
	`

	err := db.KhiladiDb.Get(&userID, query, sessionID)
	return userID, err
}


func GetUserByID(userID string) (model.User, error) {

	var user model.User

	query := `
		SELECT id
		FROM users
		WHERE id = $1 AND archived_at IS NULL
	`

	err := db.KhiladiDb.Get(&user, query, userID)
	return user, err
}

func DeleteUserSession(sessionID string)error{

	query:=`UPDATE user_session SET archived_at=NOW() where id=$1 and archived_at IS NULL`

	_,err:=db.KhiladiDb.Exec(query,sessionID);
	return  err;
}


