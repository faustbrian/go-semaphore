GOLIB ?= golib

.PHONY: check ci cohesion inventory repository-check specification-check workflows

check:
	$(GOLIB) check --all

ci: repository-check cohesion specification-check workflows check

cohesion:
	$(GOLIB) cohesion check

inventory repository-check:
	$(GOLIB) repository check

specification-check:
	$(GOLIB) specification check --online

workflows:
	$(GOLIB) workflows check
