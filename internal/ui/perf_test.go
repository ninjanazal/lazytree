package ui

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/eurico-martins/lazytree/internal/git"
	"github.com/eurico-martins/lazytree/internal/graph"
	"github.com/eurico-martins/lazytree/internal/model"
)

// TestLargeRepoLoad is an opt-in check of the M2 "smooth on a 100k+ commit
// repo" goal. Point LAZYTREE_BIGREPO at a large repository (see
// scripts/big-repo.sh for a synthetic one) and run:
//
//	LAZYTREE_BIGREPO=/path/to/repo go test ./internal/ui -run LargeRepo -v
//
// It logs timings for the first page, the full paginated load, a scroll
// pass and a search, plus heap use, and fails if any step is egregiously
// slow.
func TestLargeRepoLoad(t *testing.T) {
	repo := os.Getenv("LAZYTREE_BIGREPO")
	if repo == "" {
		t.Skip("set LAZYTREE_BIGREPO to a large repository to run")
	}
	ctx := context.Background()
	runner := &git.Runner{RepoPath: repo}

	start := time.Now()
	stream, err := git.StartLogStream(ctx, runner, "--all")
	if err != nil {
		t.Fatal(err)
	}
	p := newLogPane()
	p.setSize(120, 40)
	lt := graph.NewLayouter()

	commits, done, err := stream.Next(pageSize)
	if err != nil {
		t.Fatal(err)
	}
	p.setCommits(commits, lt.Append(commits))
	firstPage := time.Since(start)
	t.Logf("first page (%d commits) ready in %v", len(commits), firstPage)

	for !done {
		var page []model.Commit
		page, done, err = stream.Next(pageSize)
		if err != nil {
			t.Fatal(err)
		}
		p.appendCommits(page, lt.Append(page))
	}
	total := time.Since(start)
	t.Logf("full load: %d commits in %v", len(p.commits), total)

	var ms runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&ms)
	t.Logf("heap in use after load: %d MiB", ms.HeapAlloc>>20)

	// Scroll: render a frame after each of 2000 cursor moves.
	t0 := time.Now()
	for range 2000 {
		p.moveDown()
		_ = p.View(true)
	}
	perFrame := time.Since(t0) / 2000
	t.Logf("scroll: %v per move+frame", perFrame)

	t0 = time.Now()
	p.setSearch("module 42")
	t.Logf("search over %d commits: %v (%d matches)", len(p.commits), time.Since(t0), p.search.count)

	if firstPage > 3*time.Second {
		t.Errorf("first page took %v, want < 3s", firstPage)
	}
	if perFrame > 20*time.Millisecond {
		t.Errorf("scroll frame took %v, want < 20ms", perFrame)
	}
}
