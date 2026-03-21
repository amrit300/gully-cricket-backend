t) error {

	var teamLimit int

	err := db.QueryRow(`
	SELECT max_teams_per_match
	FROM users
	WHERE id=$1
	`, userID).Scan(&teamLimit)

	if err != nil {
		return fmt.Errorf("failed to fetch user plan")
	}

	var teamsToday int

	err = db.QueryRow(`
	SELECT COUNT(*)
	FROM teams
	WHERE user_id=$1
	AND created_at >= CURRENT_DATE
	`, userID).Scan(&teamsToday)

	if err != nil {
		return fmt.Errorf("failed to count teams today")
	}

	if teamsToday >= teamLimit {
		return fmt.Errorf("daily team limit reached")
	}

	return nil
}
func joinContest(c *fiber.Ctx) error {

	type Request struct {
		UserID    int `json:"user_id"`
		TeamID    int `json:"team_id"`
		ContestID int `json:"contest_id"`
	}

	var req Request

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request",
		})
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	/* =========================
	   DUPLICATE CHECK
	========================= */

	var exists int
	err = tx.QueryRow(`
	SELECT 1 FROM contest_entries
	WHERE contest_id=$1 AND team_id=$2
	`, req.ContestID, req.TeamID).Scan(&exists)

	if err == nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "already joined",
		})
	}

	/* =========================
	   LOCK CONTEST
	========================= */

	var filled, total int

	err = tx.QueryRow(`
	SELECT filled_spots, total_spots
	FROM contests
	WHERE id=$1
	FOR UPDATE
	`, req.ContestID).Scan(&filled, &total)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "contest query failed",
		})
	}

	if filled >= total {
		return c.Status(400).JSON(fiber.Map{
			"error": "contest full",
		})
	}

	/* =========================
	   TEAM OWNERSHIP CHECK
	========================= */

	var matchID, ownerID int

	err = tx.QueryRow(`
	SELECT match_id, user_id FROM teams WHERE id=$1
	`, req.TeamID).Scan(&matchID, &ownerID)

	if err != nil {
		return err
	}

	if ownerID != req.UserID {
		return c.Status(403).JSON(fiber.Map{
			"error": "unauthorized team access",
		})
	}

	/* =========================
	   INSERT ENTRY
	========================= */

	_, err = tx.Exec(`
	INSERT INTO contest_entries (contest_id,team_id,user_id)
	VALUES ($1,$2,$3)
	`, req.ContestID, req.TeamID, req.UserID)

	if err != nil {
		return err
	}

	/* =========================
	   INSERT LEADERBOARD
	========================= */

	_, err = tx.Exec(`
	INSERT INTO leaderboard (contest_id, match_id, team_id, points, rank)
	VALUES ($1,$2,$3,0,0)
	`, req.ContestID, matchID, req.TeamID)

	if err != nil {
		return err
	}

	/* =========================
	   UPDATE SPOTS
	========================= */

	_, err = tx.Exec(`
	UPDATE contests
	SET filled_spots = filled_spots + 1
	WHERE id=$1
	`, req.ContestID)

	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"status": "contest joined",
	})
}
func validateTeam(playerIDs []int) error {

	if len(playerIDs) != 11 {

		return fmt.Errorf("team must contain 11 players")
	}

	rows, err := db.Query(`
	SELECT team,role,credit
	FROM players
	WHERE id = ANY($1)
	`, pq.Array(playerIDs))

if err != nil {
	return err
}

defer rows.Close()

teamCount := map[string]int{}
roleCount := map[string]int{}

totalCredit := 0.0
playerCount := 0

for rows.Next() {

	var team string
	var role string
	var credit float64

	err := rows.Scan(&team, &role, &credit)

	if err != nil {
		return err
	}

	playerCount++

	teamCount[team]++
	roleCount[role]++
	totalCredit += credit
}
/* THEN VALIDATION STARTS */

if playerCount != 11 {
	return fmt.Errorf("invalid player selection")
}

	if totalCredit > 100 {

		return fmt.Errorf("credit limit exceeded")
	}

	for _, count := range teamCount {

		if count > 7 {

			return fmt.Errorf("max 7 players allowed from one team")
		}
	}

	if roleCount["WK"] < 1 || roleCount["WK"] > 4 {

		return fmt.Errorf("invalid wicketkeeper count")
	}

	if roleCount["BAT"] < 3 || roleCount["BAT"] > 6 {

		return fmt.Errorf("invalid batsman count")
	}

	if roleCount["ALL"] < 1 || roleCount["ALL"] > 4 {

		return fmt.Errorf("invalid allrounder count")
	}

	if roleCount["BOWL"] < 3 || roleCount["BOWL"] > 6 {

		return fmt.Errorf("invalid bowler count")
	}

	return nil
}

func getLeaderboard(c *fiber.Ctx) error {

contestID, err := strconv.Atoi(c.Params("contest_id"))

if err != nil {
	return c.Status(400).JSON(fiber.Map{
		"error":"invalid contest id",
	})
}

rows, err := db.Query(`
SELECT team_id, points, rank
FROM leaderboard
WHERE match_id=$1
ORDER BY rank ASC
`, contestID)

if err != nil {
	return c.Status(500).JSON(fiber.Map{
		"error":"database query failed",
	})
}

defer rows.Close()

var leaderboard []Entry

for rows.Next() {

	var e Entry

	err := rows.Scan(
		&e.TeamID,
		&e.Points,
		&e.Rank,
	)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":"row scan failed",
		})
	}

	leaderboard = append(leaderboard,e)
}

return c.JSON(leaderboard)
}

func updateLeaderboard(c *fiber.Ctx) error {

	type Request struct {
		MatchID int `json:"match_id"`
	}

	var req Request

	if err := c.BodyParser(&req); err != nil {
		return err
	}

	_, err := db.Exec(`
	UPDATE leaderboard l
	SET rank = r.rank
	FROM (
		SELECT team_id,
		RANK() OVER (ORDER BY points DESC) as rank
		FROM leaderboard
		WHERE match_id=$1
	) r
	WHERE l.team_id=r.team_id
	`, req.MatchID)

	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"status":"leaderboard updated",
	})
}

func processMatchEvent(c *fiber.Ctx) error {

	type Event struct {
		PlayerID int `json:"player_id"`
		Event    string `json:"event"`
		Value    int `json:"value"`
	}

	var e Event

	if err := c.BodyParser(&e); err != nil {
		return err
	}

	points := 0

	switch e.Event {

	case "run":
		points = e.Value

	case "four":
		points = 1

	case "six":
		points = 2

	case "wicket":
		points = 25

	case "catch":
		points = 8

	case "stumping":
		points = 12
	}

	_, err := db.Exec(`
	UPDATE players
	SET fantasy_points = fantasy_points + $1
	WHERE id=$2
	`, points, e.PlayerID)

	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"status": "event processed",
	})
}

func updateTeamPoints(c *fiber.Ctx) error {

	type Request struct {
		MatchID int `json:"match_id"`
	}

	var req Request

	if err := c.BodyParser(&req); err != nil {
		return err
	}

	rows, err := db.Query(`
	SELECT 
	  t.id,
	  COALESCE(SUM(
	    CASE 
	      WHEN tp.player_id = t.captain_player_id THEN p.fantasy_points * 2
	      WHEN tp.player_id = t.vice_captain_player_id THEN p.fantasy_points * 1.5
	      ELSE p.fantasy_points
	    END
	  ),0) as total
	FROM teams t
	JOIN team_players tp ON tp.team_id = t.id
	JOIN players p ON p.id = tp.player_id
	WHERE t.match_id = $1
	GROUP BY t.id
	`, req.MatchID)

	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {

		var teamID int
		var points float64

		err := rows.Scan(&teamID, &points)
		if err != nil {
			return err
		}

		_, err = db.Exec(`
		UPDATE teams
		SET total_points = $1
		WHERE id = $2
		`, points, teamID)

		if err != nil {
			return err
		}
	}

	return c.JSON(fiber.Map{
		"status": "team points updated",
	})
}
func leaderboardWorker() {

	for {

		time.Sleep(10 * time.Second)

		rows, err := db.Query(`
		SELECT DISTINCT contest_id, match_id
		FROM leaderboard
		`)
		if err != nil {
			log.Println("worker query error:", err)
			continue
		}

		for rows.Next() {

			var contestID int
			var matchID int

			if err := rows.Scan(&contestID, &matchID); err != nil {
				log.Println("scan error:", err)
				continue
			}

			/* =========================
			   STEP 1 → UPDATE TEAM POINTS
			========================= */

			_, err = db.Exec(`
			UPDATE teams t
			SET total_points = sub.points
			FROM (
				SELECT 
				  t2.id as team_id,
				  COALESCE(SUM(
				    CASE 
				      WHEN tp.player_id = t2.captain_player_id THEN p.fantasy_points * 2
				      WHEN tp.player_id = t2.vice_captain_player_id THEN p.fantasy_points * 1.5
				      ELSE p.fantasy_points
				    END
				  ),0) as points
				FROM teams t2
				JOIN team_players tp ON tp.team_id = t2.id
				JOIN players p ON p.id = tp.player_id
				WHERE t2.match_id = $1
				GROUP BY t2.id
			) sub
			WHERE t.id = sub.team_id
			AND t.match_id = $1
			`, matchID)

			if err != nil {
				log.Println("team update error:", err)
				continue
			}

			/* =========================
			   STEP 2 → SYNC LEADERBOARD POINTS
			========================= */

			_, err = db.Exec(`
			UPDATE leaderboard l
			SET points = t.total_points
			FROM teams t
			WHERE l.team_id = t.id
			AND l.contest_id = $1
			`, contestID)

			if err != nil {
				log.Println("leaderboard sync error:", err)
				continue
			}

			/* =========================
			   STEP 3 → CORRECT RANKING (FIXED)
			========================= */

			_, err = db.Exec(`
			UPDATE leaderboard l
			SET rank = r.rank
			FROM (
				SELECT 
				  team_id,
				  DENSE_RANK() OVER (
					ORDER BY points DESC, team_id ASC
				  ) as rank
				FROM leaderboard
				WHERE contest_id = $1
			) r
			WHERE l.team_id = r.team_id
			AND l.contest_id = $1
			`, contestID)

			if err != nil {
				log.Println("leaderboard rank error:", err)
				continue
			}
		}

		if err := rows.Err(); err != nil {
			log.Println("row iteration error:", err)
		}

		rows.Close()
	}
}
func syncPlayers(matchID int, externalMatchID string) error {

	players, err := internal.FetchPlayersFromCricAPI(externalMatchID)
	if err != nil {
		return err
	}

	for _, p := range players {

		name := fmt.Sprintf("%v", p["name"])

		role := "BAT" // fallback

		if r, ok := p["role"]; ok {
			role = fmt.Sprintf("%v", r)
		}

		team := "Unknown"

		if t, ok := p["team"]; ok {
			team = fmt.Sprintf("%v", t)
		}

		_, err := db.Exec(`
		INSERT INTO players (name, team, role, credit, match_id)
		VALUES ($1,$2,$3,8.5,$4)
		ON CONFLICT DO NOTHING
		`,
			name,
			team,
			role,
			matchID,
		)

		if err != nil {
			log.Println("PLAYER INSERT ERROR:", err)
		}
	}

	return nil
}
