GOLIB ?= golib

.PHONY: source-docs-check check check-full ci cohesion inventory repository-check specification-check workflows

check: source-docs-check
	$(GOLIB) check --local --all

check-full: specification-check source-docs-check
	$(GOLIB) check --all

ci: repository-check cohesion workflows check source-docs-check

cohesion:
	$(GOLIB) cohesion check

inventory repository-check:
	$(GOLIB) repository check

specification-check:
	$(GOLIB) specification check --online

workflows:
	$(GOLIB) workflows check

source-docs-check:
	$(GOLIB) source-docs check --json
