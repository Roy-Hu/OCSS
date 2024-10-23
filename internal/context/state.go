package context

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

// func CustomAlgo() StatesDiagram {
// 	diagram := StatesDiagram{
// 		Var: make(map[string]interface{}),
// 	}

// 	diagram.Var["timer1"] = time.NewTicker(5 * time.Second)
// 	diagram.Var["timer2"] = time.NewTicker(5 * time.Second)

// 	diagram.States["State1"] = State{
// 		Triggers: []func() bool{
// 			func() bool {
// 				ticker := diagram.Var["timer2"].(*time.Ticker)
// 				<-ticker.C
// 				return true
// 			},

// 			// return []byte {
// 			// 	"PERIOD": 5,
// 			// }
// 		},
// 		Actions: []func() string{
// 			func() string {
// 				for i := 0; i < len(ocssContext.Nodes["OCS_EDGE"].Connections)-1; i++ {
// 					hardware.Nodes["OCS_EDGE"].Connections[i] = hardware.Nodes["OCS_EDGE"].Connections[i+1]
// 				}
// 				hardware.Nodes["OCS_EDGE"].Connections[len(hardware.Nodes["OCS_EDGE"].Connections)-1] = hardware.Nodes["OCS_EDGE"].Connections[0]

// 				return "State1"
// 			},
// 		},
// 	}

// 	// Triggers: []func() bool{
// 	// 	func() bool {
// 	// 		for {
// 	// 			tol := <-traffic.LinkTraffics["Link_Server1_OCS1"].Total

// 	// 			if tol > 100 {
// 	// 				return true
// 	// 			}
// 	// 		}
// 	// 	},
// 	// },

// 	diagram.States["State2"] = State{
// 		Triggers: []func() bool{
// 			func() bool {
// 				ticker := diagram.Var["timer2"].(*time.Ticker)
// 				<-ticker.C
// 				return true
// 			},
// 		},
// 		Actions: []func() string{
// 			func() string {
// 				for i := 0; i < len(hardware.Nodes["OCS_CORE"].Connections)-1; i++ {
// 					rndPort := rand.Intn(hardware.Nodes["OCS_CORE"].PortNum)

// 					for rndPort == hardware.Nodes["OCS_CORE"].Connections[i] {
// 						rndPort = rand.Intn(hardware.Nodes["OCS_CORE"].PortNum)
// 					}

// 					hardware.Nodes["OCS_CORE"].Connections[i] = rndPort
// 				}

// 				return "State2"
// 			},
// 		},
// 	}

// 	return diagram
// }
