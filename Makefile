.PHONY: build dev-ui build-ui test vet lint clean

build: build-ui
	go build -o prompt-diff .

build-ui:
	npm --prefix web install
	npm --prefix web run build

dev-ui:
	npm --prefix web run dev

test:
	go test ./...

vet:
	go vet ./...

lint: vet test

clean:
	rm -f prompt-diff prompt-diff.exe report.html
