package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/etsy/hound/api"
	"github.com/etsy/hound/config"
	"github.com/etsy/hound/searcher"
	"github.com/etsy/hound/ui"
)

const gracefulShutdownSignal = syscall.SIGTERM

var (
	info_log  *log.Logger
	error_log *log.Logger
)

func registerShutdownSignal() <-chan os.Signal {
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, gracefulShutdownSignal)
	return shutdownCh
}

func runHttp(
	addr string,
	dev bool,
	cfg *config.Config,
	idx *searcher.Pool) error {
	m := http.DefaultServeMux

	h, err := ui.Content(dev, cfg)
	if err != nil {
		return err
	}

	m.Handle("/", h)
	api.Setup(m, idx)
	return http.ListenAndServe(addr, m)
}

func handleShutdown(pool *searcher.Pool, ch <-chan os.Signal) {
	pool.Shutdown(ch)
	os.Exit(0)
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	info_log = log.New(os.Stdout, "", log.LstdFlags)
	error_log = log.New(os.Stderr, "", log.LstdFlags)

	flagConf := flag.String("conf", "config.json", "")
	flagAddr := flag.String("addr", ":6080", "")
	flagDev := flag.Bool("dev", false, "")

	flag.Parse()

	var cfg config.Config
	if err := cfg.LoadFromFile(*flagConf); err != nil {
		panic(err)
	}

	host := *flagAddr
	if strings.HasPrefix(host, ":") {
		host = "localhost" + host
	}

	info_log.Printf("running server at http://%s...\n", host)
	idx, ok, err := searcher.NewPool(&cfg)
	go runHttp(*flagAddr, *flagDev, &cfg, idx)

	// It's not safe to be killed during makeSearchers, so register the
	// shutdown signal here and defer processing it until we are ready.
	shutdownCh := registerShutdownSignal()

	if err != nil {
		log.Panic(err)
	}
	if !ok {
		info_log.Println("Some repos failed to index, see output above")
	}
	idx.Index()

	handleShutdown(idx, shutdownCh)

}
