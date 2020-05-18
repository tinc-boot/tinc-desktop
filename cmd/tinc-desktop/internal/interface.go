package internal

import "context"

/*

We can't control launched tincd daemon after start.
After privilege escalation (*sudo, oascript, runas, ...) we have no more control to the process:

* impossible send signal (to rooted process)
* impossible control by STDIN/STDOUT due to privilege escalation apps are non-redirecting pipes

So we have to control by TCP, however we can detect death by exit

```

 Desktop application
         |
         |
         |     worker for network (separate process due to privilege escalation)
         +.......+
         |       |
         |       |
   Peers +<----->|
         |       |
   Kill  +------>|
         |

```
*/
type Worker interface {
	Kill(ctx context.Context) (bool, error)
	Peers(ctx context.Context) ([]string, error)
}

type Port interface {
	API() Worker
	Error() error
	Done() <-chan struct{}
	Name() string
}

type Spawner interface {
	Spawn(network string, done chan struct{}) (Port, error)
}
