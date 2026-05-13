erDiagram
	direction TB
	USERS {
		int user_id PK ""  
		string name  ""  
		string phone_number UK ""  
		string password  ""  
		datetime created_at  ""  
	}

	SESSIONS {
		int session_id PK ""  
		int user_id FK "-> USERS.user_id"  
		string token ""  
		string refresh_token ""  
		datetime created_at ""  
		datetime expires_at ""  
		datetime last_used_at ""  
		datetime revoked_at ""  
		string ip_address ""  
		string user_agent ""  
	}   

	MATCHES {
		int match_id PK ""  
		int team1_id FK "-> TEAMS.team_id"  
		int  team2_id FK "-> TEAMS.team_id"  
		int host_id FK "-> USERS.user_id"  
		int umpire_id FK "-> USERS.user_id"  
		datetime match_date  ""  
		int total_overs  ""  
		int toss_winner_team_id FK "-> TEAMS.team_id"  
		string toss_decision  ""  
		int winner_team_id FK "-> TEAMS.team_id"  
		string status  ""  
	}

	TEAMS {
		int team_id PK ""  
		string team_name  ""  
		int captain_id FK "-> PLAYERS.player_id"  
		int total_players  ""  
	}

	INNINGS {
		int innings_id PK ""  
		int match_id FK "-> MATCHES.match_id"  
		int innings_number  ""  
		int batting_team_id FK "-> TEAMS.team_id"  
		int bowling_team_id FK "-> TEAMS.team_id"  
		int total_runs  ""  
		int total_wickets  ""  
		float total_overs  ""  
		string status  ""  
	}

	PLAYERS {
		int player_id PK ""  
		string player_name ""  
		string role ""  
		int team_id FK "-> TEAMS.team_id"  
	}

	BALLS {
		int ball_id PK ""  
		int innings_id FK "-> INNINGS.innings_id"  
		int over_number  ""  
		int ball_number  ""  
		int striker_id FK "-> PLAYERS.player_id"  
		int non_striker_id FK "-> PLAYERS.player_id"  
		int bowler_id FK "-> PLAYERS.player_id"  
		int runs_scored  ""  
		int extras_runs  ""  
		string extra_type  ""  
		boolean is_wicket  ""  
		string dismissal_type  ""  
	    int dismissed_player_id FK "-> PLAYERS.player_id"  
	}

	PLAYER_MATCH_STATS {
		int stat_id PK ""  
		int match_id FK "-> MATCHES.match_id"  
		int player_id FK "-> PLAYERS.player_id"  
		int team_id FK "-> TEAMS.team_id"  
		string batting_status  ""  
		int runs_scored  ""  
		int balls_faced  ""  
		int fours  ""  
		int sixes  ""  
		float strike_rate  ""  
		float overs_bowled  ""  
		int maiden_overs  ""  
		int runs_given  ""  
		int wickets_taken  ""  
		float economy  ""  
	}