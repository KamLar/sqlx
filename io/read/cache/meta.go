package cache

type Meta struct {
	SQL        string
	Args       []byte
	Type       string
	Signature  string
	TimeToLive int
	Fields     []*Field

	url string
}