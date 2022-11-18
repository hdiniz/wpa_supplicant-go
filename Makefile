GOLANGCI_LINT_BIN=golangci-lint

TARGETS=

EXAMPLES=wifi-command wifi-events wifi-scan

EXAMPLE_BUILD_TARGETS=$(addprefix example-, $(EXAMPLES))
TARGETS += $(EXAMPLE_BUILD_TARGETS)

EXAMPLE_CLEAN_TARGETS=$(addprefix clean-example-, $(EXAMPLES))
TARGETS += $(EXAMPLE_CLEAN_TARGETS)

all: examples
.PHONY: all

examples: $(EXAMPLE_BUILD_TARGETS)
.PHONY: examples

clean: clean-examples
.PHONY = clean

clean-examples: $(EXAMPLE_CLEAN_TARGETS)
.PHONY = clean-examples

targets:
	@$(foreach target,$(TARGETS),echo $(target);)
.PHONY: targets

$(EXAMPLE_BUILD_TARGETS): example-%:
	go build -o bin/$(@:example-%=%) ./examples/$(@:example-%=%)/*.go
.PHONY = $(EXAMPLE_BUILD_TARGETS)

$(EXAMPLE_CLEAN_TARGETS): clean-example-%:
	rm -f bin/$(@:clean-example-%=%)
.PHONY = $(EXAMPLE_CLEAN_TARGETS)

test:
	go test ./...
.PHONY: test

lint:
	$(GOLANGCI_LINT_BIN) run
.PHONY: lint

vet:
	go vet ./...
.PHONY: vet

check: vet lint
.PHONY: check

require-arg-%:
	@: $(if $(value $*),,$(error required arg $* is undefined))

.PHONY: release
release: require-arg-TAG test check
	git tag $(TAG)
	git push -u origin $(TAG)



