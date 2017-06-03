package searcher

import (
	"github.com/etsy/hound/config"
	"log"
	"os"
	"time"
	"github.com/etsy/db/vcs-f47965ee76287ac9096d2200b88882933d54f755/searcher"
	"github.com/etsy/hound/index"
)

type Pool struct {
	Searchers map[string]*Searcher
}

// Make a searcher for each Repo in the Config. This function kind of has a notion
// of partial errors. First, if the error returned is non-nil then a fatal error has
// occurred and no other return values are valid. If an error occurs that is specific
// to a particular searcher, that searcher will not be present in the searcher map and
// will have an error entry in the error map.
func (pool *Pool) start(cfg *config.Config) (map[string]error, error) {
	errs := map[string]error{}

	refs, err := findExistingRefs(cfg.DbPath)
	if err != nil {
		return nil, err
	}

	lim := makeLimiter(cfg.MaxConcurrentIndexers)

	for name, repo := range cfg.Repos {
		s, err := newSearcher(cfg.DbPath, name, repo, refs, lim)
		if err != nil {
			log.Print(err)
			errs[name] = err
			continue
		}

		pool.Searchers[name] = s
	}

	if err := refs.removeUnclaimed(); err != nil {
		return nil, err
	}

	// after all the repos are in good shape, we start their polling
	for _, s := range pool.Searchers {
		s.begin()
	}

	return errs, nil
}

func MakePool(cfg *config.Config) (*Pool, bool, error) {
	pool := Pool{make(map[string]*Searcher)}
	// Ensure we have a dbpath
	if _, err := os.Stat(cfg.DbPath); err != nil {
		if err := os.MkdirAll(cfg.DbPath, os.ModePerm); err != nil {
			return nil, false, err
		}
	}

	errs, err := pool.start(cfg)
	if err != nil {
		return nil, false, err
	}

	if len(errs) > 0 {
		log.Fatal(errs)
	}

	return &pool, true, nil
}

func (pool *Pool) Shutdown(shutdownCh <-chan os.Signal) {
	<-shutdownCh
	log.Printf("Graceful shutdown requested...")
	for _, s := range pool.Searchers {
		s.Stop()
	}

	for _, s := range pool.Searchers {
		s.Wait()
	}
}

/**
 * Searches all repos in parallel.
 */
func (pool *Pool) SearchAll(
	query string,
	opts *index.SearchOptions,
	repos []string,
	filesOpened *int,
	duration *int) (map[string]*index.SearchResponse, error) {

	startedAt := time.Now()

	n := len(repos)

	// use a buffered channel to avoid routine leaks on errs.
	ch := make(chan *SearchResponse, n)
	for _, repo := range repos {
		go func(repo string) {
			fms, err := pool.Searchers[repo].Search(query, opts)
			ch <- &SearchResponse{repo, fms, err}
		}(repo)
	}

	res := map[string]*index.SearchResponse{}
	for i := 0; i < n; i++ {
		r := <-ch
		if r.Err != nil {
			return nil, r.Err
		}

		if r.Res.Matches == nil {
			continue
		}

		res[r.Repo] = r.Res
		*filesOpened += r.Res.FilesOpened
	}

	*duration = int(time.Now().Sub(startedAt).Seconds() * 1000)

	return res, nil
}