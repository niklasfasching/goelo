package goelo

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math"
	"strings"
)

type ELO struct {
	K float64 // K represents volatilty of the score, i.e. impact of an indiviual game. chess uses 20.
	N float64 // N points more ~= 90% win chance. chess uses 400.
}

func (e ELO) Expected(aScore, bScore float64) (aExpected, bExpected float64) {
	aExpected = 1.0 / (1.0 + math.Pow(10, -((aScore-bScore)/e.N)))
	return aExpected, 1 - aExpected
}

func (e ELO) NewRatings(aOld, bOld, aResult float64) (aNew, bNew float64) {
	aExpected, bExpected := e.Expected(aOld, bOld)
	aNew = aOld + e.K*(aResult-aExpected)
	bNew = bOld + e.K*((1-aResult)-bExpected)
	return aNew, bNew
}

func main() {
	games, err := read()
	if err != nil {
		log.Fatal(err)
	}
	// elo(games)
	bt(games)
}

func elo(games []*Game) {
	e := ELO{N: 400, K: 20}
	scores := map[string]float64{}
	score := func(id string) (team, p1, p2 float64) {
		parts := strings.Split(id, " :: ")
		team, ok := scores[id]
		if !ok {
			team = 1500
		}
		p1, ok = scores[parts[0]]
		if !ok {
			p1 = 1500
		}
		p2, ok = scores[parts[1]]
		if !ok {
			p2 = 1500
		}
		return team, p1, p2
	}

	update := func(a, b string, aOld, bOld, aResult float64) (aNew, bNew float64) {
		aNew, bNew = e.NewRatings(aOld, bOld, aResult)
		scores[a] = aNew
		scores[b] = bNew
		log.Printf("%s - %s: (%.0f) %.0f -> %.0f :: %.0f -> %.0f", a, b, aResult, aOld, aNew, bOld, bNew)
		return aNew, bNew
	}
	for _, g := range games {
		aParts, bParts := strings.Split(g.Team_a, " :: "), strings.Split(g.Team_b, " :: ")
		aTeam, a1, a2 := score(g.Team_a)
		bTeam, b1, b2 := score(g.Team_b)

		aResult := 0.0
		if g.Goals_a > g.Goals_b {
			aResult = 1.0
		}
		a1, a2 = update(aParts[0], aParts[1], a1, a2, 0.5)
		b1, b2 = update(bParts[0], bParts[1], b1, b2, 0.5)

		a1, b1 = update(aParts[0], bParts[0], a1, b1, aResult)
		a1, b2 = update(aParts[0], bParts[1], a1, b2, aResult)
		a2, b1 = update(aParts[1], bParts[0], a2, b1, aResult)
		a2, b2 = update(aParts[1], bParts[1], a2, b2, aResult)

		scores["wins"+aParts[0]] += aResult
		scores["wins"+aParts[1]] += aResult
		scores["wins"+bParts[0]] += 1 - aResult
		scores["wins"+bParts[1]] += 1 - aResult

		scores["losses"+aParts[0]] += 1 - aResult
		scores["losses"+aParts[1]] += 1 - aResult
		scores["losses"+bParts[0]] += aResult
		scores["losses"+bParts[1]] += aResult

		aTeam, bTeam = update(g.Team_a, g.Team_b, aTeam, bTeam, aResult)
		// correct for team skill? i.e. if part of a better team the
	}

	for k, wins := range scores {
		if strings.HasPrefix(k, "wins") {
			k = k[len("wins"):]
			losses := scores["losses"+k]
			scores["%"+k] = wins / (wins + losses)
			scores["games"+k] = (wins + losses)
		}
	}

	bs, _ := json.MarshalIndent(scores, "", "  ")
	ioutil.WriteFile("elo_scores.json", bs, 0644)

}

func read() (games []*Game, error error) {
	bs, err := ioutil.ReadFile("games.json")
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(bs, &games); err != nil {
		return nil, err
	}
	bs, err = ioutil.ReadFile("users.json")
	if err != nil {
		return nil, err
	}
	users := map[string]string{}
	if err := json.Unmarshal(bs, &users); err != nil {
		return nil, err
	}
	for _, g := range games {
		a, b := strings.Split(g.Team_a, " :: "), strings.Split(g.Team_b, " :: ")
		g.Team_a = users[a[0]] + " :: " + users[a[1]]
		g.Team_b = users[b[0]] + " :: " + users[b[1]]
	}
	return games, nil
}

type Game struct {
	Team_a  string
	Team_b  string
	Goals_a int
	Goals_b int
}
