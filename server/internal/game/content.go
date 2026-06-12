package game

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// NPC is a non-player character as defined in npcs.json. Tier 0 of the NPC
// system (proposal §13.4): a fixed position and a pool of static lines, one
// returned at random per conversation. Memory, reactions, and the AI
// npc-director come in later tiers.
type NPC struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	X     float64  `json:"x"`
	Y     float64  `json:"y"`
	Lines []string `json:"lines"`
}

//go:embed npcs.json
var npcsJSON []byte

// loadNPCs parses the embedded content file. The file is contributor-edited
// (see docs/lore/README.md), so validate enough to fail loudly at startup —
// and in CI via TestEmbeddedContentLoads — rather than mysteriously in game.
func loadNPCs() ([]NPC, error) {
	var content struct {
		NPCs []NPC `json:"npcs"`
	}
	if err := json.Unmarshal(npcsJSON, &content); err != nil {
		return nil, fmt.Errorf("npcs.json is not valid JSON: %w", err)
	}
	seen := map[string]bool{}
	for _, n := range content.NPCs {
		if n.ID == "" || n.Name == "" {
			return nil, fmt.Errorf("npc %q: id and name are required", n.ID)
		}
		if seen[n.ID] {
			return nil, fmt.Errorf("npc %q: duplicate id", n.ID)
		}
		seen[n.ID] = true
		if len(n.Lines) == 0 {
			return nil, fmt.Errorf("npc %q: needs at least one line", n.ID)
		}
		if n.X < 0 || n.X > WorldW || n.Y < 0 || n.Y > WorldH {
			return nil, fmt.Errorf("npc %q: position (%v, %v) outside world bounds", n.ID, n.X, n.Y)
		}
	}
	return content.NPCs, nil
}
