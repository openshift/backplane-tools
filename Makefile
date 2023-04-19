BUILD_DIR:=./bin

build: clean
	mkdir -p ${BUILD_DIR}
	go build -o "${BUILD_DIR}/backplane-tools" main.go

clean:
	rm -rf "${BUILD_DIR}"

test:
	echo "TODO"
