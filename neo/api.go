package neo

import (
	"go/token"

	"github.com/podhmo/reflect-shape/metadata"
)

type Config struct {
	IncludeComments    bool
	IncludeArgNames    bool
	IncludeGoTestFiles bool

	DocTruncationSize int

	extractor *Extractor
	lookup    *metadata.Lookup
}

var (
	DocTruncationSize = 10
)

func (c *Config) Extract(ob interface{}) *Shape {
	if c.DocTruncationSize == 0 {
		c.DocTruncationSize = DocTruncationSize
	}

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
