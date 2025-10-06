package main

type chunkedCode struct {
	Line    uint64
	Content string
}

type file struct {
	Name  string
	Lines []string
	EOF   uint64
}
type index struct {
	Files []file
}
