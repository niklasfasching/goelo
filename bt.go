package goelo

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math"
	"strings"
)

type Player struct {
	ID      string
	Mu      float64
	SigmaSq float64
}

type Team []*Player

func (t Team) Mu() (mu float64) {
	for _, p := range t {
		mu += p.Mu
	}
	return mu
}

func (t Team) SigmaSq() (sigmaSq float64) {
	for _, p := range t {
		sigmaSq += p.SigmaSq
	}
	return sigmaSq
}

type BradleyTerry struct {
	Mu      float64 // 25
	SigmaSq float64 // (mu/3)^2
	Beta    float64 // sigma/2
	Gamma   float64 // 1 / nTeams (i.e. always 0.5 here)
}

// ~ pg 279 https://www.csie.ntu.edu.tw/~cjlin/papers/online_ranking/online_journal.pdf

// scoreA: 1: teamA win, 0.5: draw, 0: teamB win

// the skill change of the team is partitioned onto the players by their o2
// diff between teams follows logistic distribution

func (bt BradleyTerry) Update(teamA, teamB Team, scoreA float64) {
	// ~ o2 of game
	ciq := math.Sqrt(teamA.SigmaSq() + teamB.SigmaSq() + 2*bt.Beta*bt.Beta)

	// piq = probability that teami beats teamq (i.e. teamA beats teamB)
	piq := 1 / (1 + math.Exp((teamB.Mu()-teamA.Mu())/ciq))

	omegaA := teamA.SigmaSq() / ciq * (scoreA - piq) // variance * difference to expected outcome
	deltaA := bt.Gamma * (teamA.SigmaSq() / (ciq * ciq)) * piq * (1 - piq)
	log.Println(teamA, teamB, scoreA)
	updatePlayers(teamA, omegaA, deltaA)

	omegaB := teamB.SigmaSq() / ciq * ((1 - scoreA) - (1 - piq))
	deltaB := bt.Gamma * (teamB.SigmaSq() / (ciq * ciq)) * (1 - piq) * piq
	updatePlayers(teamB, omegaB, deltaB)
}

// fetch("localhost/query=SELECT * FROM")

func updatePlayers(t Team, omega, delta float64) {
	for _, p := range t {
		log.Println(p.ID, p.Mu, p.SigmaSq/t.SigmaSq()*omega)
		p.Mu += p.SigmaSq / t.SigmaSq() * omega
		p.SigmaSq *= math.Sqrt(math.Max(1-p.SigmaSq/t.SigmaSq()*delta, 0.0001))
	}
}

func bt(games []*Game) {
	mu, sigmaSq, beta, gamma := 1000.0, 30.0, 50.0, 1.0/2.0
	bt := BradleyTerry{Mu: mu, SigmaSq: sigmaSq, Beta: beta, Gamma: gamma}
	teams, players := map[string]Team{}, map[string]*Player{}
	player := func(id string) *Player {
		if p, ok := players[id]; ok {
			return p
		}
		players[id] = &Player{id, mu, sigmaSq}
		return players[id]
	}

	team := func(id string) Team {
		if t, ok := teams[id]; ok {
			return t
		}
		parts := strings.Split(id, " :: ")
		teams[id] = Team{player(parts[0]), player(parts[1])}
		return teams[id]
	}
	// ~ 40% prediction error
	for _, g := range games {
		scoreA := 0.0
		if g.Goals_a > g.Goals_b {
			scoreA = 1.0
		}
		bt.Update(team(g.Team_a), team(g.Team_b), scoreA)
	}

	// evaluate(games, teams)
	bs, _ := json.MarshalIndent(players, "", "  ")
	ioutil.WriteFile("bt_players.json", bs, 0644)

}

// // for matchmaking: assign teams such that skill is as even as possible
// // i.e. a b c d -- ab:cd, ac:bd, ad:bc
// func evaluate(games []*Game, teams map[string]Team) {
// 	for _, g := range games {
// 		teamA, teamB := teams[g.Team_a], teams[g.Team_b]
// 		if teamA.Mu() > teamB.Mu() {
// 			if g.Goals_a > g.Goals_b {
// 				log.Println("correct", g.Team_a, g.Team_b)
// 			} else {
// 				log.Printf("incorrect: %s VS %s - %d:%d (%.1f:%.1f)", g.Team_a, g.Team_b, g.Goals_a, g.Goals_b, teamA.Mu(), teamB.Mu())
// 			}
// 		}
// 	}
// }
