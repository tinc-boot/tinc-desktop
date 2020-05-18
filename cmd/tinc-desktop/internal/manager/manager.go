package manager

import (
	"github.com/tinc-boot/tinc-desktop/cmd/tinc-desktop/internal"
	"log"
	"sync"
)

type Manager struct {
	Spawner internal.Spawner
	workers map[string]internal.Port
	lock    sync.Mutex
}

func (mgr *Manager) Find(name string) internal.Port {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	return mgr.workers[name]
}

func (mgr *Manager) Names() []string {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	var ans []string
	for k := range mgr.workers {
		ans = append(ans, k)
	}
	return ans
}

func (mgr *Manager) SpawnSudoContext(name string) (internal.Port, error) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	if wp, ok := mgr.workers[name]; ok {
		return wp, nil
	}

	if mgr.workers == nil {
		mgr.workers = make(map[string]internal.Port)
	}

	done := make(chan struct{})

	wp, err := mgr.Spawner.Spawn(name, done)
	if err != nil {
		close(done)
		return nil, err
	}

	mgr.workers[name] = wp

	go func() {
		<-done
		if err := wp.Error(); err != nil {
			log.Println(name, err)
		}
		mgr.lock.Lock()
		delete(mgr.workers, name)
		mgr.lock.Unlock()
	}()

	return wp, nil
}
