package models

type BatsmanRow struct {
	Name          string  `json:"name"`
	Runs          int     `json:"runs"`
	Balls         int     `json:"balls"`
	Fours         int     `json:"fours"`
	Sixes         int     `json:"sixes"`
	SR            string  `json:"sr"`
	IsActive      bool    `json:"isActive,omitempty"`
	IsNotOut      bool    `json:"isNotOut"`
	DismissalType *string `json:"dismissalType,omitempty"`
	BowlerName    *string `json:"bowlerName,omitempty"`
	FielderName   *string `json:"fielderName,omitempty"`
}

type BowlerRow struct {
	Name     string `json:"name"`
	Overs    string `json:"overs"`
	Maidens  int    `json:"maidens"`
	Runs     int    `json:"runs"`
	Wickets  int    `json:"wickets"`
	Econ     string `json:"econ"`
	IsActive bool   `json:"isActive,omitempty"`
}

type TopBatsman struct {
	Name  string `json:"name"`
	Score string `json:"score"`
}

type TopBowler struct {
	Name    string `json:"name"`
	Figures string `json:"figures"`
}

type ExtrasSummary struct {
	Wides   int `json:"wides" db:"wides"`
	NoBalls int `json:"noBalls" db:"noballs"`
	Byes    int `json:"byes" db:"byes"`
	LegByes int `json:"legByes" db:"legbyes"`
	Total   int `json:"total" db:"total"`
}

type InningsData struct {
	TeamName      string       `json:"teamName"`
	TeamShort     string       `json:"teamShort"`
	Runs          int          `json:"runs"`
	Wickets       int          `json:"wickets"`
	Overs         string       `json:"overs"`
	Batting       []BatsmanRow `json:"batting"`
	Bowling       []BowlerRow  `json:"bowling"`
	YetToBat      []string     `json:"yetToBat"`
	FallOfWickets []string     `json:"fallOfWickets"`
	TopBatsmen    []TopBatsman `json:"topBatsmen"`
	TopBowlers    []TopBowler  `json:"topBowlers"`
	Extras        ExtrasSummary `json:"extras"`
}

type MatchDetail struct {
	ID               string       `json:"id"`
	TeamA            string       `json:"teamA"`
	TeamB            string       `json:"teamB"`
	TeamAShort       string       `json:"teamAShort"`
	TeamBShort       string       `json:"teamBShort"`
	Status           string       `json:"status"`
	StatusText       string       `json:"statusText"`
	ResultText       *string      `json:"resultText,omitempty"`
	Date             string       `json:"date"`
	Time             string       `json:"time"`
	TotalOvers       int          `json:"totalOvers"`
	TossWinner       string       `json:"tossWinner"`
	TossDecision     string       `json:"tossDecision"`
	Umpire           string       `json:"umpire"`
	Host             string       `json:"host"`
	PlayerOfTheMatch *struct {
		Name      string `json:"name"`
		TeamShort string `json:"teamShort"`
		Stats     string `json:"stats"`
	} `json:"playerOfTheMatch,omitempty"`
	Innings1 InningsData  `json:"innings1"`
	Innings2 *InningsData `json:"innings2,omitempty"`
}

type BallRow struct {
	StrikerID    string  `db:"striker_id"`
	NonStrikerID *string `db:"non_striker_id"`
	BowlerID     string  `db:"bowler_id"`
}

type BatStat struct {
	PlayerID      string  `db:"player_id"`
	Name          string  `db:"player_name"`
	Runs          int     `db:"runs_scored"`
	Balls         int     `db:"balls_faced"`
	Fours         int     `db:"fours"`
	Sixes         int     `db:"sixes"`
	IsNotOut      bool    `db:"is_not_out"`
	DismissalType *string `db:"dismissal_type"`
	DismissedBy   *string `db:"bowler_name"`
	FielderName   *string `db:"fielder_name"`
}

type BowlStat struct {
	PlayerID    string  `db:"player_id"`
	Name        string  `db:"player_name"`
	OversBowled float64 `db:"overs_bowled"`
	RunsGiven   int     `db:"runs_given"`
	Wickets     int     `db:"wickets_taken"`
}

type TopBatsmanRow struct {
	Name  string `db:"player_name"`
	Runs  int    `db:"runs_scored"`
	Balls int    `db:"balls_faced"`
}

type TopBowlerRow struct {
	Name    string  `db:"player_name"`
	Wickets int     `db:"wickets_taken"`
	Runs    int     `db:"runs_given"`
	Overs   float64 `db:"overs_bowled"`
}

type PlayerNameRow struct {
	PlayerID string `db:"player_id"`
	Name     string `db:"player_name"`
}


