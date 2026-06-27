package game

import "time"

// GatherRange is how close (world units) a player must stand to a node to
// gather it. Enforced server-side; the client prompt is advisory, exactly
// like TalkRange.
const GatherRange = 48.0

// respawnByType is how long a depleted node of each type stays gone before
// it comes back. loadResources rejects any node whose type is absent here,
// so adding a node type is a deliberate two-line change, not a silent
// "respawns never" bug.
var respawnByType = map[string]time.Duration{
	"driftwood":  30 * time.Second,
	"scrap_iron": 45 * time.Second, // rarer than driftwood, so a touch slower
}

// resourceNode is the hub's runtime copy of a content node: the static
// definition plus the live availability state. Only the hub goroutine
// touches these fields, so no locking is needed.
type resourceNode struct {
	def       ResourceNode
	available bool
	respawnAt time.Time
}

func newResourceNodes(defs []ResourceNode) []*resourceNode {
	nodes := make([]*resourceNode, len(defs))
	for i, d := range defs {
		nodes[i] = &resourceNode{def: d, available: true}
	}
	return nodes
}
