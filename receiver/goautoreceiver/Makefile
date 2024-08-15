include ../../Makefile.Common

.PHONY: goautogen
goautogen:
	go mod download
	GO_AUTO_PATH=$$(go list -m -f '{{.Dir}}' "go.opentelemetry.io/auto") && \
	sudo chmod -R +w $$GO_AUTO_PATH && \
	cd $$GO_AUTO_PATH && \
	make generate
