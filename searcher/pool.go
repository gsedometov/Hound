package searcher

import (
	"github.com/etsy/hound/config"
	"log"
	"os"
	"time"
	"github.com/etsy/hound/index"
	"github.com/etsy/hound/status"
)

type Pool struct {
	Searchers map[string]*Searcher
	lim limiter
	dbpath string
	Status status.Code
	cfg *config.Config
}

// Make a searcher for each Repo in the Config. This function kind of has a notion
// of partial errors. First, if the error returned is non-nil then a fatal error has
// occurred and no other return values are valid. If an error occurs that is specific
// to a particular searcher, that searcher will not be present in the searcher map and
// will have an error entry in the error map.
func (pool *Pool) start() (map[string]error, error) {
	errs := map[string]error{}
	cfg := pool.cfg

	refs, err := findExistingRefs(pool.cfg.DbPath)
	if err != nil {
		return nil, err
	}

	for name, repo := range cfg.Repos {
		s, err := newSearcher(cfg.DbPath, name, repo, refs, pool.lim)
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

func NewPool (cfg *config.Config) (*Pool, bool, error) {
	pool := &Pool{
		make(map[string]*Searcher),
		makeLimiter(cfg.MaxConcurrentIndexers),
		cfg.DbPath,
		status.Starting,
		cfg,
	}
	// Ensure we have a dbpath
	if _, err := os.Stat(cfg.DbPath); err != nil {
		if err := os.MkdirAll(cfg.DbPath, os.ModePerm); err != nil {
			pool.Status = status.Error
			return pool, false, err
		}
	}
	return pool, true, nil
}

func (pool *Pool) Index() (*Pool, bool, error) {
	pool.Status = status.Indexing
	errs, err := pool.start()
	if err != nil {
		pool.Status = status.Error
		return nil, false, err
	}

	if len(errs) > 0 {
		pool.Status = status.Error
		log.Fatal(errs)
	}

	pool.Status = status.Ready
	return pool, true, nil
}

func (pool *Pool) Shutdown(shutdownCh <-chan os.Signal) {
	<-shutdownCh
	pool.Status = status.Stopping
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

func (pool *Pool) AddRepos (repos map[string]*config.Repo){
	for name, repo := range repos {
		refs, err := findExistingRefs(pool.dbpath)
		s, err := newSearcher(pool.dbpath, name, repo, refs, pool.lim)
		if err != nil {
			log.Print(err)
			continue
		}

		pool.Searchers[name] = s
		s.begin()
	}
}