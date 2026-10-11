GOLIB ?= golib

.PHONY: check check-full ci cohesion inventory repository-check specification-check workflows

check:
	$(GOLIB) check --local --all

check-full: specification-check
	$(GOLIB) check --all

ci: repository-check cohesion workflows check

cohesion:
	$(GOLIB) cohesion check

inventory repository-check:
	$(GOLIB) repository check

specification-check:
	$(GOLIB) specification check --online

workflows:
	$(GOLIB) workflows check
