BIN_DIR:=./bin

.PHONY: build
build: clean
	mkdir -p ${BIN_DIR}
	go build -mod=mod -o "${BIN_DIR}/backplane-tools" main.go

.PHONY: clean
clean:
	rm -rf "${BIN_DIR}"

.PHONY: test
test:
	go test ./pkg/... ./cmd/... -count=1 -mod=mod
