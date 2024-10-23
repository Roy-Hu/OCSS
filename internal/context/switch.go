package context

type Switch struct {
	Name  string
	Id    int
	Ports map[int]*ConnectedTo
}
