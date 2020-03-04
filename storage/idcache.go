package storage

type IDCache map[string]int

func (c *IDCache) Get(s string) int {
	if s == "" {
		return 0
	}
	id, _ := (*c)[s]
	return id
}
