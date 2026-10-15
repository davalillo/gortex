package store_sqlite

import (
	"path/filepath"
	"testing"

	"github.com/zzet/gortex/internal/graph"
)

// mirroredFTSStore seeds a store with two repos' node rows written through
// the lean mirror path, plus one FTS doc filed WITHOUT a mirror row (the
// orphan shape the scope predicate must keep excluding).
func mirroredFTSStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "mirror.sqlite"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	mirror, ok := graph.Store(s).(graph.FTSNodeMirror)
	if !ok {
		t.Fatalf("store must implement graph.FTSNodeMirror")
	}
	nodes := []*graph.Node{
		{ID: "app/a.go::Alpha", Kind: graph.KindFunction, Name: "AlphaWidget", FilePath: "a.go", RepoPrefix: "app"},
		{ID: "beta/b.go::BetaWidget", Kind: graph.KindType, Name: "BetaWidget", FilePath: "b.go", RepoPrefix: "beta"},
	}
	if err := mirror.MirrorFTSNodes(nodes); err != nil {
		t.Fatalf("MirrorFTSNodes: %v", err)
	}

	docs := []graph.SymbolFTSItem{
		{NodeID: "app/a.go::Alpha", Tokens: "alpha widget"},
		{NodeID: "beta/b.go::BetaWidget", Tokens: "beta widget"},
		{NodeID: "orphan::ghost::Ghost", Tokens: "ghost widget"},
	}
	if err := s.BatchUpsertSymbolFTS(docs); err != nil {
		t.Fatalf("BatchUpsertSymbolFTS: %v", err)
	}
	return s
}

// TestMirrorFTSNodes_FilesDocsByMirrorPrefix: the lean mirror alone must be
// sufficient for FTS filing — docs whose node row was mirrored get filed
// under the mirrored repo_prefix and pass the scoped narrow, exactly as if
// the full AddBatch path had written them.
func TestMirrorFTSNodes_FilesDocsByMirrorPrefix(t *testing.T) {
	s := mirroredFTSStore(t)

	appHits, err := s.SearchSymbolsRepoScoped("widget", []string{"app"}, 20)
	if err != nil {
		t.Fatalf("scoped(app): %v", err)
	}
	if len(appHits) != 1 || appHits[0].NodeID != "app/a.go::Alpha" {
		t.Fatalf("scoped(app) = %+v, want exactly Alpha", appHits)
	}

	betaHits, err := s.SearchSymbolsRepoScoped("widget", []string{"beta"}, 20)
	if err != nil {
		t.Fatalf("scoped(beta): %v", err)
	}
	if len(betaHits) != 1 || betaHits[0].NodeID != "beta/b.go::BetaWidget" {
		t.Fatalf("scoped(beta) = %+v, want exactly BetaWidget", betaHits)
	}
}

// TestMirrorFTSNodes_Tier0ExactNameShortCircuit: the engine's exact-name
// tier-0 reads the mirrored rows — a symbol-kind exact name short-circuits
// with the dominant score, mirroring the full-node-write behaviour.
func TestMirrorFTSNodes_Tier0ExactNameShortCircuit(t *testing.T) {
	s := mirroredFTSStore(t)

	hits, err := s.SearchSymbolsRepoScoped("AlphaWidget", []string{"app"}, 20)
	if err != nil {
		t.Fatalf("scoped: %v", err)
	}
	if len(hits) != 1 || hits[0].NodeID != "app/a.go::Alpha" || hits[0].Score != 100.0 {
		t.Fatalf("tier-0 exact-name = %+v, want the 100.0 short-circuit on Alpha", hits)
	}
}

// TestMirrorFTSNodes_UnmirroredDocsStayOrphans: a doc upserted without a
// mirror row files under the empty prefix and must stay excluded from
// scoped searches (the #25 orphan semantics, unchanged by the lean path).
func TestMirrorFTSNodes_UnmirroredDocsStayOrphans(t *testing.T) {
	s := mirroredFTSStore(t)

	hits, err := s.SearchSymbolsRepoScoped("ghost", []string{"app"}, 20)
	if err != nil {
		t.Fatalf("scoped: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("scoped search leaked the unmirrored doc: %+v", hits)
	}

	unscoped, err := s.SearchSymbols("ghost", 20)
	if err != nil {
		t.Fatalf("unscoped: %v", err)
	}
	found := false
	for _, h := range unscoped {
		if h.NodeID == "orphan::ghost::Ghost" {
			found = true
		}
	}
	if !found {
		t.Fatalf("unscoped search lost the unmirrored doc: %+v", unscoped)
	}
}

// TestMirrorFTSNodes_UpsertByID: re-mirroring the same ID updates the row in
// place (last write wins), matching AddBatch's upsert semantics.
func TestMirrorFTSNodes_UpsertByID(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "mirror2.sqlite"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	mirror, ok := graph.Store(s).(graph.FTSNodeMirror)
	if !ok {
		t.Fatalf("store must implement graph.FTSNodeMirror")
	}

	first := &graph.Node{ID: "app/a.go::Alpha", Kind: graph.KindFunction, Name: "OldName", FilePath: "a.go", RepoPrefix: "app"}
	if err := mirror.MirrorFTSNodes([]*graph.Node{first}); err != nil {
		t.Fatalf("MirrorFTSNodes(1st): %v", err)
	}
	second := &graph.Node{ID: "app/a.go::Alpha", Kind: graph.KindType, Name: "AlphaWidget", FilePath: "a.go", RepoPrefix: "app"}
	if err := mirror.MirrorFTSNodes([]*graph.Node{second}); err != nil {
		t.Fatalf("MirrorFTSNodes(2nd): %v", err)
	}

	hits, err := s.SearchSymbolsRepoScoped("AlphaWidget", []string{"app"}, 20)
	if err != nil {
		t.Fatalf("scoped: %v", err)
	}
	if len(hits) != 1 || hits[0].NodeID != first.ID || hits[0].Score != 100.0 {
		t.Fatalf("re-mirror must satisfy the new exact name: %+v", hits)
	}
}
