TARGETS := $(shell ls scripts)

.dapper:
	@echo Downloading dapper
	@curl -sL https://releases.rancher.com/dapper/latest/dapper-`uname -s`-`uname -m` > .dapper.tmp
	@@chmod +x .dapper.tmp
	@./.dapper.tmp -v
	@mv .dapper.tmp .dapper

$(TARGETS): .dapper
	./.dapper $@

e2e: e2e-sonobuoy
	$(MAKE) e2e-verify

clean:
	rm -rvf ./bin ./dist

.DEFAULT_GOAL := ci

.PHONY: $(TARGETS) e2e clean
