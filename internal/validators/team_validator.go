package validators

import (
	"errors"
)

type Player struct {
	ID     int
	Role   string
	Credit float64
}

func ValidateTeam(players []Player, captainID int, viceCaptainID int) error {

	if len(players) != 11 {
		return errors.New("team must have exactly 11 players")
	}

	totalCredit := 0.0

	roleCount := map[string]int{
		"batsman": 0,
		"bowler":  0,
		"allrounder": 0,
		"wicketkeeper": 0,
	}

	playerMap := make(map[int]bool)

	for _, p := range players {

		if playerMap[p.ID] {
			return errors.New("duplicate player selected")
		}
		playerMap[p.ID] = true

		totalCredit += p.Credit

		roleCount[p.Role]++
	}

	if totalCredit > 100 {
		return errors.New("credit limit exceeded")
	}

	if captainID == viceCaptainID {
		return errors.New("captain and vice captain must be different")
	}

	if !playerMap[captainID] || !playerMap[viceCaptainID] {
		return errors.New("captain/vice must be in team")
	}

	return nil
}
