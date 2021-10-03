RUN_CMD := go run ../cmd/reflect-shape-dump 
SHELL := bash

default: 00 01 02

00: data
	$(RUN_CMD) github.com/podhmo/reflect-shape/_examples/$(shell echo $@*).Hello | tee data/$@.output
01: data
	$(RUN_CMD) github.com/podhmo/reflect-shape/_examples/$(shell echo $@*).Foo.Hello | tee data/$@.output
02: data
	$(RUN_CMD) github.com/podhmo/reflect-shape/_examples/$(shell echo $@*).HelloFunc | tee data/$@.output

data:
	mkdir -p $@

clean:
	rm -r -f data .reflect-shape*
.PHONY: clean
