package neo

import (
	"go/token"

	"github.com/podhmo/reflect-shape/metadata"
)

type Config struct {
	IncludeComments bool
	IncludeArgNames bool

	extractor *Extractor
	lookup    *metadata.Lookup
}

func (c *Config) Extract(ob interface{}) *Shape {
	if c.lookup == nil {
		c.lookup = metadata.NewLookup(token.NewFileSet())
	}
	if c.extractor == nil {
		c.extractor = &Extractor{
			Config:   c,
			Lookup:   c.lookup,
			seen:     map[ID]*Shape{},
			packages: map[string]*Package{},
		}
	}
	return c.extractor.Extract(ob)
}
