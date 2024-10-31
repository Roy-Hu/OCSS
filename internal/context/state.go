package context

import (
	"context"
	"time"
)

type State struct {
	Triggers  []func(ctx context.Context) bool `yaml:"-"`
	Actions   []func() string                  `yaml:"-"`
	InitState bool
}

// assume we have the following functions
func GetTraffic() Traffic {
	return Traffic{}
}

func SetupStates(user *UserView) map[string]*State {
	states := make(map[string]*State)

	states["State1"] = &State{
		Triggers: []func(ctx context.Context) bool{
			func(ctx context.Context) bool {
				ticker := time.NewTicker(5 * time.Second)
				defer ticker.Stop()

				select {
				case <-ticker.C:
					return true
				case <-ctx.Done():
					return false
				}
			},
		},
		Actions: []func() string{
			func() string {
				first_out_port := user.OCSs["ocs_edge"].Conn.Out_port[0]
				for i := 0; i < len(user.OCSs["ocs_edge"].Conn.Out_port)-1; i++ {
					user.OCSs["ocs_edge"].Conn.Out_port[i] = user.OCSs["ocs_edge"].Conn.Out_port[i+1]
				}
				user.OCSs["ocs_edge"].Conn.Out_port[len(user.OCSs["ocs_edge"].Conn.Out_port)-1] = first_out_port

				return "State1"
			},
		},
		InitState: true,
	}

	return states
}
