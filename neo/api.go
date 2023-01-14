package neo

import (
	"go/token"

	"github.com/podhmo/reflect-shape/metadata"
)

type Config struct {
	IncludeComments    bool
	IncludeArgNames    bool
	IncludeGoTestFiles bool

	extractor *Extractor
	lookup    *metadata.Lookup
}

func (c *Config) Extract(ob interface{}) *Shape {
	if c.lookup == nil {
		c.lookup = metadata.NewLookup(token.NewFileSet())
		c.lookup.IncludeGoTestFiles = c.IncludeGoTestFiles
		// c.lookup.IncludeUnexported = c.Inc
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
