// +build darwin linux

package spawners

import (
	"github.com/tinc-boot/tinc-desktop/cmd/tinc-desktop/internal"
	"os"
)

func SelectSpawner(configLocation string) internal.Spawner {
	if os.Geteuid() == 0 {
		return &SameProcess{ConfigLocation: configLocation}
	}
	return &SubProcess{ConfigLocation: configLocation}
}
