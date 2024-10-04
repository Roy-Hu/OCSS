package context

import (
	"math/rand"
	"time"
)

type State struct {
	Triggers  []func() bool
	Actions   []func() string
	InitState bool
}

type StatesDiagram struct {
	Var    map[string]interface{} // allows any type
	States map[string]State
}

// assume we have the following functions
func GetTraffic() Traffic {
	return Traffic{}
}

func GetHardware() Hardware {
	return Hardware{}
}

func CustomAlgo() StatesDiagram {
	diagram := StatesDiagram{
		Var: make(map[string]interface{}),
	}

	hardware := GetHardware()

	diagram.Var["timer1"] = time.NewTicker(5 * time.Second)
	diagram.Var["timer2"] = time.NewTicker(5 * time.Second)

	diagram.States["State1"] = State{
		Triggers: []func() bool{
			func() bool {
				ticker := diagram.Var["timer2"].(*time.Ticker)
				<-ticker.C
				return true
			},

			// return []byte {
			// 	"PERIOD": 5,
			// }
		},
		Actions: []func() string{
			func() string {
				for i := 0; i < len(hardware.Nodes["OCS_EDGE"].Connections)-1; i++ {
					hardware.Nodes["OCS_EDGE"].Connections[i] = hardware.Nodes["OCS_EDGE"].Connections[i+1]
				}
				hardware.Nodes["OCS_EDGE"].Connections[len(hardware.Nodes["OCS_EDGE"].Connections)-1] = hardware.Nodes["OCS_EDGE"].Connections[0]

				// return []byte {
				// 	"Name": "OCS_EDGE",
				// 	"Connections": {
				// 	  "3": NODES[OCS_EDGE][Connections][0],
				// 	  "0": NODES[OCS_EDGE][Connections][1],
				// 	  "1": NODES[OCS_EDGE][Connections][2],
				// 	  "2": NODES[OCS_EDGE][Connections][3],
				// 	},
				//  "NextState": "State1",
				// }

				// return []byte {
				// 	"Name": "OCS_EDGE",
				//  "Connections": {
				//       SHIFT: 1
				//   }
				//  "NextState": "State1",
				// }
				return "State1"
			},
		},
	}

	// Triggers: []func() bool{
	// 	func() bool {
	// 		for {
	// 			tol := <-traffic.LinkTraffics["Link_Server1_OCS1"].Total

	// 			if tol > 100 {
	// 				return true
	// 			}
	// 		}
	// 	},
	// },

	// return []byte {
	// 	"Name": "Link_Server1_OCS1",
	//  "Total": {
	//      gt: 100
	// 	},
	diagram.States["State2"] = State{
		Triggers: []func() bool{
			func() bool {
				ticker := diagram.Var["timer2"].(*time.Ticker)
				<-ticker.C
				return true
			},
		},
		Actions: []func() string{
			func() string {
				for i := 0; i < len(hardware.Nodes["OCS_CORE"].Connections)-1; i++ {
					rndPort := rand.Intn(hardware.Nodes["OCS_CORE"].PortNumOut)

					for rndPort == hardware.Nodes["OCS_CORE"].Connections[i] {
						rndPort = rand.Intn(hardware.Nodes["OCS_CORE"].PortNumOut)
					}

					hardware.Nodes["OCS_CORE"].Connections[i] = rndPort
				}

				return "State2"
			},
		},

		// return []byte {
		// 	"Name": "OCS_CORE",
		// 	"Connections": {
		// 	  "0": RND: {4},
		// 	  "1": RND: {4},
		// 	  "2": RND: {4},
		// 	  "3": RND: {4},
		// 	},
		//  "NextState": "State1",
		// }
	}

	return diagram
}
