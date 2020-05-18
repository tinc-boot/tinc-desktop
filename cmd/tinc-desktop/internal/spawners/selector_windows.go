package spawners

import (
	"github.com/tinc-boot/tinc-desktop/cmd/tinc-desktop/internal"
)

func SelectSpawner(configLocation string) internal.Spawner {
	return &SameProcess{ConfigLocation: configLocation}
}
